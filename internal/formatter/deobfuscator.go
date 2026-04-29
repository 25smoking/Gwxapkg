package formatter

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja/ast"
	"github.com/dop251/goja/parser"
	"github.com/dop251/goja/token"
)

const deobfuscationTimeout = 300 * time.Millisecond

var (
	obfuscatedIdentifierPattern = regexp.MustCompile(`_0x[a-f0-9]{3,}`)
)

var errDeobfuscationTimeout = errors.New("反混淆执行超时")

var analyzeJavaScriptCore = analyzeJavaScript

// DeobfuscationResult 反混淆分析结果
type DeobfuscationResult struct {
	Content       []byte
	IsObfuscated  bool
	Score         int
	Techniques    []string
	Status        string
	StaticDecoded bool
	RestoredCalls int
}

type bootstrapAnalysis struct {
	source     string
	program    *ast.Program
	arrays     map[string]struct{}
	decoders   map[string]struct{}
	statements []ast.Statement
	score      int
	techniques map[string]struct{}
}

type replacement struct {
	start   int
	end     int
	literal string
}

type runtimeState struct {
	vm          *goja.Runtime
	decoderFns  map[string]goja.Callable
	runtimeText map[string][]string
}

// AnalyzeJavaScript 分析并尝试还原常见 JS 混淆
func AnalyzeJavaScript(input []byte, filePath string) (result *DeobfuscationResult, err error) {
	result = &DeobfuscationResult{
		Content: append([]byte(nil), input...),
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			result = &DeobfuscationResult{
				Content:    append([]byte(nil), input...),
				Techniques: []string{"analysis-panic"},
				Status:     "skipped",
			}
			err = nil
		}
	}()

	return analyzeJavaScriptCore(input, filePath)
}

func analyzeJavaScript(input []byte, filePath string) (*DeobfuscationResult, error) {
	original := string(input)
	staticContent, staticTechniques, staticChanged := decodeStaticJavaScript(original)

	result := &DeobfuscationResult{
		Content:       []byte(staticContent),
		Techniques:    dedupeStrings(staticTechniques),
		StaticDecoded: staticChanged,
	}

	analysis := analyzeBootstrap(staticContent)
	result.Score += analysis.score
	result.Techniques = dedupeStrings(append(result.Techniques, mapKeys(analysis.techniques)...))

	if staticChanged {
		result.Score += 18
	}
	if obfuscatedIdentifierPattern.MatchString(staticContent) {
		result.Score += 12
		result.Techniques = dedupeStrings(append(result.Techniques, "hex-identifier"))
	}

	rewritten, restoredCalls, runtimeErr := rewriteWithRuntime(staticContent, analysis)
	if restoredCalls > 0 {
		result.Content = []byte(rewritten)
		result.RestoredCalls = restoredCalls
		result.Score += min(restoredCalls*8, 40)
	}

	remainingScore := remainingObfuscationScore(string(result.Content))
	if remainingScore > 0 {
		result.Score += remainingScore
	}

	switch {
	case restoredCalls > 0 && remainingScore == 0:
		result.Status = "restored"
	case restoredCalls > 0 || staticChanged:
		result.Status = "partial"
	case result.Score >= 35:
		result.Status = "flagged"
	default:
		result.Status = ""
	}

	result.IsObfuscated = result.Status != ""
	if errors.Is(runtimeErr, errDeobfuscationTimeout) {
		result.Status = "flagged"
		result.IsObfuscated = true
		result.Techniques = dedupeStrings(append(result.Techniques, "execution-timeout"))
	}

	if result.Content == nil {
		result.Content = input
	}

	return result, nil
}

// BuildObfuscatedTag 构造混淆标签
func BuildObfuscatedTag(result *DeobfuscationResult) string {
	if result == nil || !result.IsObfuscated {
		return ""
	}
	return fmt.Sprintf("[OBFUSCATED] status=%s techniques=%s", result.Status, strings.Join(result.Techniques, ","))
}

func prependObfuscatedHeader(content []byte, result *DeobfuscationResult) []byte {
	tag := BuildObfuscatedTag(result)
	if tag == "" {
		return content
	}
	return append([]byte("/* "+tag+" */\n"), content...)
}

func decodeStaticJavaScript(source string) (string, []string, bool) {
	var out strings.Builder
	out.Grow(len(source))

	inSingle, inDouble, inTemplate := false, false, false
	inLineComment, inBlockComment := false, false
	inTemplateExprDepth := 0
	changed := false
	techniques := make([]string, 0)

	for i := 0; i < len(source); {
		ch := source[i]

		if inLineComment {
			out.WriteByte(ch)
			i++
			if ch == '\n' {
				inLineComment = false
			}
			continue
		}
		if inBlockComment {
			out.WriteByte(ch)
			i++
			if ch == '*' && i < len(source) && source[i] == '/' {
				out.WriteByte(source[i])
				i++
				inBlockComment = false
			}
			continue
		}

		if !inSingle && !inDouble && !inTemplate && ch == '/' && i+1 < len(source) {
			if source[i+1] == '/' {
				out.WriteString("//")
				i += 2
				inLineComment = true
				continue
			}
			if source[i+1] == '*' {
				out.WriteString("/*")
				i += 2
				inBlockComment = true
				continue
			}
		}

		if inTemplate {
			if ch == '`' && inTemplateExprDepth == 0 {
				inTemplate = false
				out.WriteByte(ch)
				i++
				continue
			}
			if ch == '$' && i+1 < len(source) && source[i+1] == '{' {
				inTemplateExprDepth++
				out.WriteString("${")
				i += 2
				continue
			}
			if ch == '}' && inTemplateExprDepth > 0 {
				inTemplateExprDepth--
				out.WriteByte(ch)
				i++
				continue
			}
		}

		if inSingle || inDouble || (inTemplate && inTemplateExprDepth == 0) {
			if ch == '\\' && i+1 < len(source) {
				if source[i+1] == 'x' && i+3 < len(source) {
					if decoded, ok := decodeEscapedHex(source[i+2 : i+4]); ok {
						out.WriteRune(decoded)
						i += 4
						changed = true
						techniques = append(techniques, "hex-literal")
						continue
					}
				}
				if source[i+1] == 'u' && i+5 < len(source) {
					if decoded, ok := decodeEscapedUnicode(source[i+2 : i+6]); ok {
						out.WriteRune(decoded)
						i += 6
						changed = true
						techniques = append(techniques, "unicode-literal")
						continue
					}
				}
			}

			if ch == '\'' && inSingle {
				inSingle = false
			}
			if ch == '"' && inDouble {
				inDouble = false
			}
			out.WriteByte(ch)
			i++
			continue
		}

		switch ch {
		case '\'':
			inSingle = true
			out.WriteByte(ch)
			i++
			continue
		case '"':
			inDouble = true
			out.WriteByte(ch)
			i++
			continue
		case '`':
			inTemplate = true
			out.WriteByte(ch)
			i++
			continue
		}

		if ch == '0' && i+2 < len(source) && (source[i+1] == 'x' || source[i+1] == 'X') {
			j := i + 2
			for j < len(source) && isHexDigit(source[j]) {
				j++
			}
			if j > i+2 && isNumberBoundary(source, i-1) && isNumberBoundary(source, j) {
				value, err := strconv.ParseInt(source[i:j], 0, 64)
				if err == nil {
					out.WriteString(strconv.FormatInt(value, 10))
					i = j
					changed = true
					techniques = append(techniques, "hex-number")
					continue
				}
			}
		}

		out.WriteByte(ch)
		i++
	}

	return out.String(), dedupeStrings(techniques), changed
}

func decodeEscapedHex(value string) (rune, bool) {
	parsed, err := strconv.ParseUint(value, 16, 8)
	if err != nil {
		return 0, false
	}
	return rune(parsed), true
}

func decodeEscapedUnicode(value string) (rune, bool) {
	parsed, err := strconv.ParseUint(value, 16, 16)
	if err != nil {
		return 0, false
	}
	return rune(parsed), true
}

func isHexDigit(ch byte) bool {
	return ('0' <= ch && ch <= '9') || ('a' <= ch && ch <= 'f') || ('A' <= ch && ch <= 'F')
}

func isNumberBoundary(source string, idx int) bool {
	if idx < 0 || idx >= len(source) {
		return true
	}
	ch := source[idx]
	return !(ch == '_' || ch == '$' || ch == '.' || ('0' <= ch && ch <= '9') || ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z'))
}

func analyzeBootstrap(source string) *bootstrapAnalysis {
	analysis := &bootstrapAnalysis{
		source:     source,
		arrays:     make(map[string]struct{}),
		decoders:   make(map[string]struct{}),
		techniques: make(map[string]struct{}),
	}

	program, err := parser.ParseFile(nil, "", source, 0)
	if err != nil {
		return analysis
	}
	if program == nil {
		return analysis
	}
	analysis.program = program

	for _, statement := range program.Body {
		if statement == nil {
			continue
		}
		switch node := statement.(type) {
		case *ast.VariableStatement:
			if markStringArrayStatement(node, analysis) {
				analysis.statements = append(analysis.statements, statement)
			}
		case *ast.FunctionDeclaration:
			if markDecoderFunction(node.Function, analysis) {
				analysis.statements = append(analysis.statements, statement)
			}
		}
	}

	for _, statement := range program.Body {
		if statement == nil {
			continue
		}
		switch node := statement.(type) {
		case *ast.VariableStatement:
			if markDecoderVarStatement(node, analysis) {
				analysis.statements = append(analysis.statements, statement)
			}
		case *ast.ExpressionStatement:
			if markRotationExpression(node, analysis, source) {
				analysis.statements = append(analysis.statements, statement)
			}
		}
	}

	analysis.statements = dedupeStatements(analysis.statements)
	return analysis
}

func markStringArrayStatement(statement *ast.VariableStatement, analysis *bootstrapAnalysis) bool {
	if statement == nil || analysis == nil {
		return false
	}
	matched := false
	for _, binding := range statement.List {
		if binding == nil {
			continue
		}
		identifier, ok := binding.Target.(*ast.Identifier)
		if !ok || identifier == nil {
			continue
		}
		arrayLiteral, ok := binding.Initializer.(*ast.ArrayLiteral)
		if !ok || arrayLiteral == nil {
			continue
		}
		if !isLikelyStringArray(arrayLiteral) {
			continue
		}
		analysis.arrays[identifier.Name.String()] = struct{}{}
		analysis.techniques["string-array"] = struct{}{}
		analysis.score += 35
		matched = true
	}
	return matched
}

func markDecoderVarStatement(statement *ast.VariableStatement, analysis *bootstrapAnalysis) bool {
	if statement == nil || analysis == nil {
		return false
	}
	matched := false
	for _, binding := range statement.List {
		if binding == nil {
			continue
		}
		identifier, ok := binding.Target.(*ast.Identifier)
		if !ok || identifier == nil {
			continue
		}
		switch initializer := binding.Initializer.(type) {
		case *ast.FunctionLiteral:
			if markDecoderFunctionWithName(identifier.Name.String(), initializer, analysis) {
				analysis.decoders[identifier.Name.String()] = struct{}{}
				matched = true
			}
		case *ast.Identifier:
			if _, exists := analysis.decoders[initializer.Name.String()]; exists {
				analysis.decoders[identifier.Name.String()] = struct{}{}
				matched = true
			}
		}
	}
	return matched
}

func markDecoderFunction(function *ast.FunctionLiteral, analysis *bootstrapAnalysis) bool {
	if function == nil || function.Name == nil {
		return false
	}
	return markDecoderFunctionWithName(function.Name.Name.String(), function, analysis)
}

func markDecoderFunctionWithName(name string, function *ast.FunctionLiteral, analysis *bootstrapAnalysis) bool {
	if function == nil || analysis == nil || name == "" {
		return false
	}
	if obfuscatedIdentifierPattern.MatchString(name) || functionReferencesAny(function, analysis.arrays) {
		analysis.decoders[name] = struct{}{}
		analysis.techniques["index-decoder"] = struct{}{}
		analysis.score += 24
		return true
	}
	return false
}

func markRotationExpression(statement *ast.ExpressionStatement, analysis *bootstrapAnalysis, source string) bool {
	if len(analysis.arrays) == 0 {
		return false
	}
	snippet := sliceNodeSource(source, statement)
	if snippet == "" {
		return false
	}
	if !strings.Contains(snippet, "function") {
		return false
	}
	if !containsAny(snippet, mapKeys(analysis.arrays)) {
		return false
	}
	if !(strings.Contains(snippet, "shift(") || strings.Contains(snippet, "push(") || strings.Contains(snippet, "while")) {
		return false
	}
	analysis.techniques["array-rotator"] = struct{}{}
	analysis.score += 18
	return true
}

func rewriteWithRuntime(source string, analysis *bootstrapAnalysis) (string, int, error) {
	if analysis.program == nil || (len(analysis.arrays) == 0 && len(analysis.decoders) == 0) {
		return source, 0, nil
	}

	bootstrap := buildBootstrapSource(source, analysis.statements)
	if bootstrap == "" {
		return source, 0, nil
	}

	state, err := prepareRuntime(bootstrap, analysis)
	if err != nil {
		return source, 0, err
	}

	replacements := collectReplacements(analysis.program, state, source)
	if len(replacements) == 0 {
		return source, 0, nil
	}

	slices.SortFunc(replacements, func(a, b replacement) int {
		return b.start - a.start
	})

	rewritten := source
	for _, item := range replacements {
		if item.start < 0 || item.end > len(rewritten) || item.start >= item.end {
			continue
		}
		rewritten = rewritten[:item.start] + item.literal + rewritten[item.end:]
	}

	return rewritten, len(replacements), nil
}

func prepareRuntime(bootstrap string, analysis *bootstrapAnalysis) (*runtimeState, error) {
	vm := goja.New()
	if err := runWithTimeout(vm, deobfuscationTimeout, func() error {
		_, err := vm.RunString(bootstrap)
		return err
	}); err != nil {
		return nil, err
	}

	state := &runtimeState{
		vm:          vm,
		decoderFns:  make(map[string]goja.Callable),
		runtimeText: make(map[string][]string),
	}

	for name := range analysis.decoders {
		fn, ok := goja.AssertFunction(vm.Get(name))
		if ok {
			state.decoderFns[name] = fn
		}
	}
	for name := range analysis.arrays {
		exported := vm.Get(name).Export()
		switch values := exported.(type) {
		case []interface{}:
			texts := make([]string, 0, len(values))
			for _, value := range values {
				texts = append(texts, fmt.Sprint(value))
			}
			state.runtimeText[name] = texts
		case []string:
			state.runtimeText[name] = slices.Clone(values)
		}
	}

	return state, nil
}

func collectReplacements(program *ast.Program, state *runtimeState, source string) []replacement {
	if program == nil || state == nil {
		return nil
	}
	replacements := make([]replacement, 0)
	seen := make(map[string]struct{})

	walkNode(program, func(node ast.Node) {
		switch expr := node.(type) {
		case *ast.CallExpression:
			if expr == nil {
				return
			}
			name, ok := calleeName(expr.Callee)
			if !ok {
				return
			}
			fn, exists := state.decoderFns[name]
			if !exists {
				return
			}
			args, ok := evaluateArguments(expr.ArgumentList)
			if !ok {
				return
			}
			value, err := callDecoder(state.vm, fn, args)
			if err != nil {
				return
			}
			literal := toJSLiteral(value)
			key := fmt.Sprintf("%d:%d", int(expr.Idx0()), int(expr.Idx1()))
			if _, exists := seen[key]; exists {
				return
			}
			seen[key] = struct{}{}
			replacements = append(replacements, replacement{
				start:   nodeStart(expr),
				end:     nodeEnd(expr),
				literal: literal,
			})
		case *ast.BracketExpression:
			if expr == nil {
				return
			}
			identifier, ok := expr.Left.(*ast.Identifier)
			if !ok || identifier == nil {
				return
			}
			values, exists := state.runtimeText[identifier.Name.String()]
			if !exists {
				return
			}
			indexValue, ok := evaluateStaticExpression(expr.Member)
			if !ok {
				return
			}
			index, ok := toInt(indexValue)
			if !ok || index < 0 || index >= len(values) {
				return
			}
			key := fmt.Sprintf("%d:%d", int(expr.Idx0()), int(expr.Idx1()))
			if _, exists := seen[key]; exists {
				return
			}
			seen[key] = struct{}{}
			replacements = append(replacements, replacement{
				start:   nodeStart(expr),
				end:     nodeEnd(expr),
				literal: strconv.Quote(values[index]),
			})
		}
	})

	return replacements
}

func buildBootstrapSource(source string, statements []ast.Statement) string {
	if len(statements) == 0 {
		return ""
	}

	var out strings.Builder
	for _, statement := range statements {
		if statement == nil {
			continue
		}
		snippet := strings.TrimSpace(sliceNodeSource(source, statement))
		if snippet == "" {
			continue
		}
		if containsDangerousRuntime(snippet) {
			continue
		}
		out.WriteString(snippet)
		if !strings.HasSuffix(snippet, ";") && !strings.HasSuffix(snippet, "}") && !strings.HasSuffix(snippet, ")") {
			out.WriteString(";")
		}
		out.WriteByte('\n')
	}
	return out.String()
}

func containsDangerousRuntime(snippet string) bool {
	dangerous := []string{
		"require(", "import(", "wx.request", "fetch(", "XMLHttpRequest", "setTimeout", "setInterval",
		"Page(", "App(", "Component(", "Behavior(", "module.exports", "exports.",
	}
	return containsAny(snippet, dangerous)
}

func runWithTimeout(vm *goja.Runtime, timeout time.Duration, fn func() error) error {
	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case err := <-done:
		vm.ClearInterrupt()
		return err
	case <-timer.C:
		vm.Interrupt(errDeobfuscationTimeout)
		err := <-done
		vm.ClearInterrupt()
		var interrupted *goja.InterruptedError
		if errors.As(err, &interrupted) {
			return errDeobfuscationTimeout
		}
		if err == nil {
			return errDeobfuscationTimeout
		}
		return err
	}
}

func callDecoder(vm *goja.Runtime, fn goja.Callable, args []interface{}) (goja.Value, error) {
	values := make([]goja.Value, 0, len(args))
	for _, arg := range args {
		values = append(values, vm.ToValue(arg))
	}

	var (
		result goja.Value
		err    error
	)
	runErr := runWithTimeout(vm, deobfuscationTimeout, func() error {
		result, err = fn(goja.Undefined(), values...)
		return err
	})
	if runErr != nil {
		return nil, runErr
	}
	return result, nil
}

func evaluateArguments(args []ast.Expression) ([]interface{}, bool) {
	result := make([]interface{}, 0, len(args))
	for _, arg := range args {
		value, ok := evaluateStaticExpression(arg)
		if !ok {
			return nil, false
		}
		result = append(result, value)
	}
	return result, true
}

func evaluateStaticExpression(expr ast.Expression) (interface{}, bool) {
	switch node := expr.(type) {
	case *ast.StringLiteral:
		return node.Value.String(), true
	case *ast.NumberLiteral:
		return node.Value, true
	case *ast.BooleanLiteral:
		return node.Value, true
	case *ast.NullLiteral:
		return nil, true
	case *ast.UnaryExpression:
		value, ok := evaluateStaticExpression(node.Operand)
		if !ok {
			return nil, false
		}
		number, ok := toFloat(value)
		if !ok {
			return nil, false
		}
		switch node.Operator {
		case token.MINUS:
			return -number, true
		case token.PLUS:
			return number, true
		case token.BITWISE_NOT:
			return float64(^int64(number)), true
		default:
			return nil, false
		}
	case *ast.BinaryExpression:
		left, ok := evaluateStaticExpression(node.Left)
		if !ok {
			return nil, false
		}
		right, ok := evaluateStaticExpression(node.Right)
		if !ok {
			return nil, false
		}
		return evalBinaryExpression(node.Operator, left, right)
	case *ast.SequenceExpression:
		if len(node.Sequence) == 0 {
			return nil, false
		}
		return evaluateStaticExpression(node.Sequence[len(node.Sequence)-1])
	default:
		return nil, false
	}
}

func evalBinaryExpression(operator token.Token, left, right interface{}) (interface{}, bool) {
	if operator == token.PLUS {
		if leftStr, ok := left.(string); ok {
			return leftStr + fmt.Sprint(right), true
		}
		if rightStr, ok := right.(string); ok {
			return fmt.Sprint(left) + rightStr, true
		}
	}

	leftNum, ok := toFloat(left)
	if !ok {
		return nil, false
	}
	rightNum, ok := toFloat(right)
	if !ok {
		return nil, false
	}

	switch operator {
	case token.PLUS:
		return leftNum + rightNum, true
	case token.MINUS:
		return leftNum - rightNum, true
	case token.MULTIPLY:
		return leftNum * rightNum, true
	case token.SLASH:
		if rightNum == 0 {
			return nil, false
		}
		return leftNum / rightNum, true
	case token.REMAINDER:
		if rightNum == 0 {
			return nil, false
		}
		return float64(int64(leftNum) % int64(rightNum)), true
	case token.SHIFT_LEFT:
		return float64(int64(leftNum) << uint64(int64(rightNum))), true
	case token.SHIFT_RIGHT:
		return float64(int64(leftNum) >> uint64(int64(rightNum))), true
	case token.UNSIGNED_SHIFT_RIGHT:
		return float64(uint32(leftNum) >> uint32(rightNum)), true
	case token.AND:
		return float64(int64(leftNum) & int64(rightNum)), true
	case token.OR:
		return float64(int64(leftNum) | int64(rightNum)), true
	case token.EXCLUSIVE_OR:
		return float64(int64(leftNum) ^ int64(rightNum)), true
	default:
		return nil, false
	}
}

func toFloat(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	default:
		return 0, false
	}
}

func toInt(value interface{}) (int, bool) {
	number, ok := toFloat(value)
	if !ok {
		return 0, false
	}
	intValue := int(number)
	if float64(intValue) != number {
		return 0, false
	}
	return intValue, true
}

func toJSLiteral(value goja.Value) string {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return "null"
	}
	switch exported := value.Export().(type) {
	case string:
		return strconv.Quote(exported)
	case bool:
		if exported {
			return "true"
		}
		return "false"
	default:
		if number, ok := toFloat(exported); ok {
			if float64(int64(number)) == number {
				return strconv.FormatInt(int64(number), 10)
			}
			return strconv.FormatFloat(number, 'f', -1, 64)
		}
		return strconv.Quote(value.String())
	}
}

func remainingObfuscationScore(source string) int {
	score := 0
	if obfuscatedIdentifierPattern.MatchString(source) {
		score += 15
	}
	if strings.Contains(source, "while (!![])") || strings.Contains(source, "while(!![])") {
		score += 10
	}
	if strings.Contains(source, "['push']") || strings.Contains(source, "['shift']") {
		score += 8
	}
	return score
}

func functionReferencesAny(function *ast.FunctionLiteral, names map[string]struct{}) bool {
	found := false
	walkNode(function, func(node ast.Node) {
		if found {
			return
		}
		identifier, ok := node.(*ast.Identifier)
		if !ok {
			return
		}
		if _, exists := names[identifier.Name.String()]; exists {
			found = true
		}
	})
	return found
}

func isLikelyStringArray(arrayLiteral *ast.ArrayLiteral) bool {
	if arrayLiteral == nil || len(arrayLiteral.Value) == 0 {
		return false
	}
	count := 0
	totalLength := 0
	for _, expr := range arrayLiteral.Value {
		if expr == nil {
			return false
		}
		switch value := expr.(type) {
		case *ast.StringLiteral:
			count++
			totalLength += len(value.Value.String())
		default:
			return false
		}
	}
	return count >= 3 || totalLength >= 8
}

func dedupeStatements(statements []ast.Statement) []ast.Statement {
	result := make([]ast.Statement, 0, len(statements))
	seen := make(map[string]struct{})
	for _, statement := range statements {
		if statement == nil {
			continue
		}
		key := fmt.Sprintf("%d:%d", int(statement.Idx0()), int(statement.Idx1()))
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, statement)
	}
	return result
}

func sliceNodeSource(source string, node ast.Node) string {
	if node == nil {
		return ""
	}
	start := nodeStart(node)
	end := nodeEnd(node)
	if start < 0 || end > len(source) || start >= end {
		return ""
	}
	return source[start:end]
}

func nodeStart(node ast.Node) int {
	if node == nil {
		return -1
	}
	return max(int(node.Idx0())-1, 0)
}

func nodeEnd(node ast.Node) int {
	if node == nil {
		return -1
	}
	return max(int(node.Idx1())-1, 0)
}

func calleeName(expr ast.Expression) (string, bool) {
	switch node := expr.(type) {
	case *ast.Identifier:
		if node == nil {
			return "", false
		}
		return node.Name.String(), true
	case *ast.DotExpression:
		if node == nil {
			return "", false
		}
		return node.Identifier.Name.String(), true
	default:
		return "", false
	}
}

func walkNode(node ast.Node, fn func(ast.Node)) {
	if isNilNode(node) {
		return
	}

	fn(node)
	walkStructFields(reflect.ValueOf(node), fn)
}

func isNilNode(node ast.Node) bool {
	if node == nil {
		return true
	}
	value := reflect.ValueOf(node)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func walkStructFields(value reflect.Value, fn func(ast.Node)) {
	if !value.IsValid() {
		return
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return
		}
		walkStructFields(value.Elem(), fn)
		return
	}

	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return
		}
		walkStructFields(value.Elem(), fn)
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			field := value.Field(i)
			if !field.CanInterface() {
				continue
			}
			if node, ok := field.Interface().(ast.Node); ok {
				walkNode(node, fn)
				continue
			}
			walkStructFields(field, fn)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			item := value.Index(i)
			if item.CanInterface() {
				if node, ok := item.Interface().(ast.Node); ok {
					walkNode(node, fn)
					continue
				}
			}
			walkStructFields(item, fn)
		}
	}
}

func mapKeys(values map[string]struct{}) []string {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	slices.Sort(result)
	return result
}

func containsAny(source string, values []string) bool {
	for _, value := range values {
		if value != "" && strings.Contains(source, value) {
			return true
		}
	}
	return false
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
