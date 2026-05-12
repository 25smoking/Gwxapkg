package semantic

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/dop251/goja/ast"
	"github.com/dop251/goja/parser"
)

const astRenameReportFileName = "ast_rename_map.json"
const astRenameDiffFileName = "ast_rename_diff.md"
const astRenamePatchFileName = "ast_rename.patch"
const preASTSourcesDirName = "pre_ast_sources"
const gojaASTPackagePath = "github.com/dop251/goja/ast"
const maxASTRenameFileBytes = 180 * 1024

var hexIdentifierPattern = regexp.MustCompile(`^_0x[0-9a-fA-F]+$`)

const (
	ASTRenameModeOff    = "off"
	ASTRenameModeReport = "report"
	ASTRenameModeSafe   = "safe"
	ASTRenameModeDeep   = "deep"

	ASTConfidenceHigh   = "high"
	ASTConfidenceMedium = "medium"
	ASTConfidenceLow    = "low"
)

// ASTRenameOptions 控制 AST 重命名的写回强度和审计产物。
type ASTRenameOptions struct {
	Mode          string
	GenerateDiff  bool
	GeneratePatch bool
}

// DefaultASTRenameOptions 默认走 deep，优先生成审计友好的源码视图。
func DefaultASTRenameOptions() ASTRenameOptions {
	return ASTRenameOptions{
		Mode:          ASTRenameModeDeep,
		GenerateDiff:  true,
		GeneratePatch: true,
	}
}

// ASTRenameReport 记录 AST 级变量/函数重命名的完整追溯信息。
type ASTRenameReport struct {
	GeneratedAt    string                `json:"generated_at"`
	Mode           string                `json:"mode"`
	TotalRenames   int                   `json:"total_renames"`
	CandidateCount int                   `json:"candidate_count"`
	RenamedFiles   int                   `json:"renamed_files"`
	DiffPath       string                `json:"diff_path,omitempty"`
	PatchPath      string                `json:"patch_path,omitempty"`
	Files          []ASTFileRenameReport `json:"files"`
}

// ASTFileRenameReport 描述单个 JS 文件的 AST 重命名结果。
type ASTFileRenameReport struct {
	FilePath    string          `json:"file_path"`
	Status      string          `json:"status"`
	RenameCount int             `json:"rename_count,omitempty"`
	Renames     []ASTRenameItem `json:"renames,omitempty"`
	Skipped     []ASTSkipItem   `json:"skipped,omitempty"`
	Error       string          `json:"error,omitempty"`
}

// ASTRenameItem 描述一次可追溯的 binding 重命名。
type ASTRenameItem struct {
	OriginalName     string `json:"original_name"`
	NewName          string `json:"new_name"`
	BeforeName       string `json:"before_name,omitempty"`
	AfterName        string `json:"after_name,omitempty"`
	ScopeKind        string `json:"scope_kind"`
	Reason           string `json:"reason"`
	Confidence       string `json:"confidence"`
	Applied          bool   `json:"applied"`
	SkipReason       string `json:"skip_reason,omitempty"`
	ReplacementCount int    `json:"replacement_count"`
	LineNumber       int    `json:"line_number"`
	SourceSnippet    string `json:"source_snippet,omitempty"`
}

// ASTSkipItem 描述一个未重命名 binding 的原因。
type ASTSkipItem struct {
	Name       string `json:"name"`
	ScopeKind  string `json:"scope_kind,omitempty"`
	Reason     string `json:"reason"`
	LineNumber int    `json:"line_number,omitempty"`
}

type astScope struct {
	id       int
	kind     string
	start    int
	end      int
	parent   *astScope
	children []*astScope
	bindings []*astBinding
	used     map[string]struct{}
}

type astBinding struct {
	name        string
	scope       *astScope
	offset      int
	kind        string
	initSource  string
	initExpr    ast.Expression
	paramIndex  int
	paramTotal  int
	functionSrc string
}

type astIdentifierOccurrence struct {
	name   string
	start  int
	end    int
	binder *astBinding
}

type astReplacement struct {
	start int
	end   int
	from  string
	to    string
}

type astRenameCandidate struct {
	binding      *astBinding
	target       string
	reason       string
	confidence   string
	replacements []astReplacement
}

type astFileChange struct {
	relPath string
	before  string
	after   string
	renames []ASTRenameItem
}

type astRenameContext struct {
	relPath              string
	source               string
	program              *ast.Program
	scopes               []*astScope
	bindings             []*astBinding
	occurrences          []astIdentifierOccurrence
	occurrencesByBinding map[*astBinding][]astIdentifierOccurrence
	skipOffsets          map[int]string
	shorthandByName      map[string][]int
}

// RenameIdentifiers 在已还原工程目录中对 JS 文件做 AST 级变量/函数重命名，并写出追溯报告。
func RenameIdentifiers(rootDir string, jsFiles []string) (*ASTRenameReport, error) {
	return RenameIdentifiersWithOptions(rootDir, jsFiles, DefaultASTRenameOptions())
}

// RenameIdentifiersWithOptions 在已还原工程目录中对 JS 文件做可配置的 AST 级变量/函数重命名。
func RenameIdentifiersWithOptions(rootDir string, jsFiles []string, options ASTRenameOptions) (*ASTRenameReport, error) {
	rootAbs, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("解析输出目录失败: %w", err)
	}
	options = normalizeASTRenameOptions(options)
	if len(jsFiles) == 0 {
		jsFiles, err = collectAllJSFiles(rootAbs)
		if err != nil {
			return nil, err
		}
	}

	report := &ASTRenameReport{
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		Mode:        options.Mode,
		Files:       make([]ASTFileRenameReport, 0, len(jsFiles)),
	}
	if options.Mode == ASTRenameModeOff {
		if err := writeASTRenameReport(rootAbs, report); err != nil {
			return nil, err
		}
		return report, nil
	}

	changes := make([]astFileChange, 0)
	for _, rel := range jsFiles {
		if shouldSkipASTRenameFile(rel) {
			continue
		}
		fileReport, change, err := renameIdentifiersInFile(rootAbs, rel, options)
		if err != nil {
			return nil, err
		}
		report.Files = append(report.Files, fileReport)
		report.CandidateCount += len(fileReport.Renames)
		if fileReport.RenameCount > 0 {
			report.RenamedFiles++
			report.TotalRenames += fileReport.RenameCount
		}
		if change != nil {
			changes = append(changes, *change)
		}
	}

	sort.Slice(report.Files, func(i, j int) bool {
		return report.Files[i].FilePath < report.Files[j].FilePath
	})
	if options.GenerateDiff {
		if err := writeASTRenameDiff(rootAbs, changes); err != nil {
			return nil, err
		}
		report.DiffPath = path.Join(reportDirName, astRenameDiffFileName)
	}
	if options.GeneratePatch {
		if err := writeASTRenamePatch(rootAbs, changes); err != nil {
			return nil, err
		}
		report.PatchPath = path.Join(reportDirName, astRenamePatchFileName)
	}
	if err := writeASTRenameReport(rootAbs, report); err != nil {
		return nil, err
	}
	return report, nil
}

func normalizeASTRenameOptions(options ASTRenameOptions) ASTRenameOptions {
	mode := strings.ToLower(strings.TrimSpace(options.Mode))
	switch mode {
	case ASTRenameModeOff, ASTRenameModeReport, ASTRenameModeSafe, ASTRenameModeDeep:
	default:
		mode = ASTRenameModeDeep
	}
	options.Mode = mode
	return options
}

func shouldApplyASTRename(mode, confidence string) bool {
	switch mode {
	case ASTRenameModeReport, ASTRenameModeOff:
		return false
	case ASTRenameModeDeep:
		return confidence == ASTConfidenceHigh || confidence == ASTConfidenceMedium
	default:
		return confidence == ASTConfidenceHigh
	}
}

// ASTRenameNoticeLines 返回适合命令行展示的 AST 策略提示。
func ASTRenameNoticeLines(options ASTRenameOptions) []string {
	options = normalizeASTRenameOptions(options)
	parameterLine := fmt.Sprintf("本次参数: -ast-rename=%s -ast-diff=%t -ast-patch=%t", options.Mode, options.GenerateDiff, options.GeneratePatch)
	artifactLine := buildASTArtifactNotice(options)
	switch options.Mode {
	case ASTRenameModeOff:
		return []string{
			"AST 还原策略: off（关闭变量/函数重命名）",
			parameterLine,
			"本次不会写回 AST 重命名，也不会生成新的 AST diff/patch。",
		}
	case ASTRenameModeReport:
		return []string{
			"AST 还原策略: report（只分析不写回）",
			parameterLine,
			"本次只分析候选并生成报告，不修改源码里的变量或函数名。",
		}
	case ASTRenameModeSafe:
		return []string{
			"AST 还原策略: safe（保守写回）",
			parameterLine,
			"仅写回 high 置信度命名，例如 require 别名、getApp、API 参数、request options、Promise resolve/reject。",
			"不会改 exports.xxx、对象字段、字符串、注释、WXML handler 和 wx/uni/getApp/require/module/exports 等公开或全局标识。",
			artifactLine,
		}
	default:
		return []string{
			"AST 还原策略: deep（激进写回，默认）",
			parameterLine,
			"会写回 high/medium 置信度的局部变量、函数参数和可完整追踪的局部函数名，以提升审计可读性。",
			"可能把 e/t/r/u/n 等压缩名改成 params、requestData、response、event、app 等语义名；low 置信度仍只进报告不写回。",
			"不会改 exports.xxx、对象字段、字符串、注释、WXML handler 和 wx/uni/getApp/require/module/exports 等公开或全局标识。",
			artifactLine,
		}
	}
}

func buildASTArtifactNotice(options ASTRenameOptions) string {
	artifacts := []string{"ast_rename_map.json"}
	if options.GenerateDiff {
		artifacts = append(artifacts, "ast_rename_diff.md")
	}
	if options.GeneratePatch {
		artifacts = append(artifacts, "ast_rename.patch")
	}
	return fmt.Sprintf("写回前会保留 .gwxapkg/pre_ast_sources，并生成 %s；可用 semantic -ast-rollback=true 回滚。", strings.Join(artifacts, " / "))
}

func shouldSkipASTRenameFile(rel string) bool {
	if strings.HasPrefix(rel, reportDirName+"/") {
		return true
	}
	if strings.HasPrefix(rel, "wxcomponents/") ||
		strings.HasPrefix(rel, "@babel/") ||
		strings.HasPrefix(rel, "colorui/") ||
		strings.HasPrefix(rel, "uni_modules/") {
		return true
	}
	base := path.Base(rel)
	if strings.HasSuffix(base, ".min.js") || strings.Contains(base, "vendor") {
		return true
	}
	if strings.HasPrefix(base, "module_") ||
		strings.HasPrefix(base, "ui_") ||
		strings.HasPrefix(base, "utils_") ||
		strings.HasPrefix(base, "mini_program_page") ||
		base == "uni_icon_font.js" {
		return true
	}
	return false
}

func renameIdentifiersInFile(rootDir, rel string, options ASTRenameOptions) (ASTFileRenameReport, *astFileChange, error) {
	fullPath := filepath.Join(rootDir, filepath.FromSlash(rel))
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return ASTFileRenameReport{}, nil, err
	}
	source := string(data)
	fileReport := ASTFileRenameReport{
		FilePath: rel,
		Status:   "unchanged",
	}
	if len(data) > maxASTRenameFileBytes {
		fileReport.Status = "skipped"
		fileReport.Skipped = append(fileReport.Skipped, ASTSkipItem{Reason: "large-library-file"})
		return fileReport, nil, nil
	}
	if strings.TrimSpace(source) == "" {
		fileReport.Status = "skipped"
		fileReport.Skipped = append(fileReport.Skipped, ASTSkipItem{Reason: "empty-file"})
		return fileReport, nil, nil
	}

	program, err := parser.ParseFile(nil, rel, source, parser.IgnoreRegExpErrors, parser.WithDisableSourceMaps)
	if err != nil {
		fileReport.Status = "skipped"
		fileReport.Error = err.Error()
		fileReport.Skipped = append(fileReport.Skipped, ASTSkipItem{Reason: "parse-error"})
		return fileReport, nil, nil
	}

	ctx := buildASTRenameContext(rel, source, program)
	candidates := ctx.planRenames()
	if len(candidates) == 0 {
		fileReport.Skipped = ctx.skippedItems()
		return fileReport, nil, nil
	}

	replacements := make([]astReplacement, 0)
	for _, candidate := range candidates {
		applied := shouldApplyASTRename(options.Mode, candidate.confidence)
		if applied {
			replacements = append(replacements, candidate.replacements...)
		}
		item := ASTRenameItem{
			OriginalName:     candidate.binding.name,
			NewName:          candidate.target,
			BeforeName:       candidate.binding.name,
			AfterName:        candidate.target,
			ScopeKind:        candidate.binding.scope.kind,
			Reason:           candidate.reason,
			Confidence:       candidate.confidence,
			Applied:          applied,
			ReplacementCount: len(candidate.replacements),
			LineNumber:       lineNumberAtOffset(source, candidate.binding.offset),
			SourceSnippet:    astRenameSnippet(source, candidate.binding.offset),
		}
		if !applied {
			item.SkipReason = "mode-" + options.Mode + "-does-not-apply-" + candidate.confidence
		}
		fileReport.Renames = append(fileReport.Renames, item)
	}
	next, changed := applyASTReplacements(source, replacements)
	if !changed {
		fileReport.Skipped = ctx.skippedItems()
		return fileReport, nil, nil
	}
	if err := backupPreASTSource(rootDir, rel, source); err != nil {
		return ASTFileRenameReport{}, nil, err
	}
	if err := os.WriteFile(fullPath, []byte(next), 0644); err != nil {
		return ASTFileRenameReport{}, nil, err
	}
	sort.Slice(fileReport.Renames, func(i, j int) bool {
		if fileReport.Renames[i].LineNumber != fileReport.Renames[j].LineNumber {
			return fileReport.Renames[i].LineNumber < fileReport.Renames[j].LineNumber
		}
		return fileReport.Renames[i].OriginalName < fileReport.Renames[j].OriginalName
	})
	fileReport.Status = "renamed"
	for _, rename := range fileReport.Renames {
		if rename.Applied {
			fileReport.RenameCount++
		}
	}
	fileReport.Skipped = ctx.skippedItems()
	change := &astFileChange{
		relPath: rel,
		before:  source,
		after:   next,
		renames: fileReport.Renames,
	}
	return fileReport, change, nil
}

func buildASTRenameContext(rel, source string, program *ast.Program) *astRenameContext {
	ctx := &astRenameContext{
		relPath:         rel,
		source:          source,
		program:         program,
		skipOffsets:     make(map[int]string),
		shorthandByName: make(map[string][]int),
	}
	ctx.collectScopes()
	ctx.collectBindings()
	ctx.collectSkipOffsets()
	ctx.collectOccurrences()
	ctx.resolveOccurrences()
	ctx.indexOccurrences()
	return ctx
}

func (ctx *astRenameContext) collectScopes() {
	root := &astScope{
		id:    0,
		kind:  "program",
		start: 0,
		end:   len(ctx.source),
		used:  make(map[string]struct{}),
	}
	ctx.scopes = append(ctx.scopes, root)

	nextID := 1
	seen := map[string]struct{}{
		scopeKey(root.kind, root.start, root.end): {},
	}
	addScope := func(kind string, start, end int) {
		if start < 0 || end <= start {
			return
		}
		key := scopeKey(kind, start, end)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		ctx.scopes = append(ctx.scopes, &astScope{
			id:    nextID,
			kind:  kind,
			start: start,
			end:   end,
			used:  make(map[string]struct{}),
		})
		nextID++
	}
	walkASTNode(ctx.program, func(node ast.Node) {
		switch item := node.(type) {
		case *ast.FunctionLiteral:
			if item == nil {
				return
			}
			addScope("function", astNodeStart(item), astNodeEnd(item))
		case *ast.ArrowFunctionLiteral:
			if item == nil {
				return
			}
			addScope("arrow-function", astNodeStart(item), astNodeEnd(item))
		case *ast.BlockStatement:
			if item == nil {
				return
			}
			addScope("block", astNodeStart(item), astNodeEnd(item))
		}
	})

	for _, scope := range ctx.scopes {
		if scope == root {
			continue
		}
		scope.parent = ctx.smallestContainingScope(scope)
		if scope.parent == nil {
			scope.parent = root
		}
		scope.parent.children = append(scope.parent.children, scope)
	}
}

func scopeKey(kind string, start, end int) string {
	return fmt.Sprintf("%s:%d:%d", kind, start, end)
}

func (ctx *astRenameContext) smallestContainingScope(target *astScope) *astScope {
	var parent *astScope
	for _, scope := range ctx.scopes {
		if scope == target {
			continue
		}
		if scope.start > target.start || scope.end < target.end {
			continue
		}
		if scope.start == target.start && scope.end == target.end && scope.kind != "program" {
			continue
		}
		if parent == nil || scopeRange(scope) < scopeRange(parent) ||
			(scopeRange(scope) == scopeRange(parent) && parent.kind == "program" && scope.kind != "program") {
			parent = scope
		}
	}
	return parent
}

func scopeRange(scope *astScope) int {
	if scope == nil {
		return 0
	}
	return scope.end - scope.start
}

func (ctx *astRenameContext) collectBindings() {
	walkASTNode(ctx.program, func(node ast.Node) {
		switch item := node.(type) {
		case *ast.FunctionLiteral:
			ctx.collectFunctionBindings(item)
		case *ast.ArrowFunctionLiteral:
			ctx.collectArrowFunctionBindings(item)
		case *ast.VariableStatement:
			scope := ctx.functionScopeAt(astNodeStart(item))
			ctx.collectBindingList(item.List, scope, "var", 0, 0, "")
		case *ast.LexicalDeclaration:
			scope := ctx.innermostScopeAt(astNodeStart(item))
			ctx.collectBindingList(item.List, scope, strings.ToLower(item.Token.String()), 0, 0, "")
		case *ast.ForLoopInitializerVarDeclList:
			scope := ctx.functionScopeAt(astNodeStart(item))
			ctx.collectBindingList(item.List, scope, "var", 0, 0, "")
		case *ast.ForLoopInitializerLexicalDecl:
			scope := ctx.innermostScopeAt(astNodeStart(item))
			ctx.collectBindingList(item.LexicalDeclaration.List, scope, strings.ToLower(item.LexicalDeclaration.Token.String()), 0, 0, "")
		case *ast.ForIntoVar:
			scope := ctx.functionScopeAt(astNodeStart(item))
			ctx.collectBinding(item.Binding, scope, "var", 0, 0, "")
		case *ast.ForDeclaration:
			scope := ctx.innermostScopeAt(astNodeStart(item))
			ctx.collectBindingTarget(item.Target, nil, scope, "for-declaration", 0, 0, "")
		case *ast.CatchStatement:
			scope := ctx.innermostScopeAt(astNodeStart(item))
			ctx.collectBindingTarget(item.Parameter, nil, scope, "catch-param", 0, 0, "")
		}
	})
}

func (ctx *astRenameContext) collectFunctionBindings(function *ast.FunctionLiteral) {
	if function == nil {
		return
	}
	scope := ctx.scopeByExactRange("function", astNodeStart(function), astNodeEnd(function))
	if scope == nil {
		scope = ctx.innermostScopeAt(astNodeStart(function))
	}
	functionSrc := sliceASTNodeSource(ctx.source, function)
	if function.Name != nil {
		parentScope := ctx.functionScopeAt(astNodeStart(function))
		ctx.addBinding(function.Name, nil, parentScope, "function", -1, 0, functionSrc)
	}
	ctx.collectParameterList(function.ParameterList, scope, functionSrc)
}

func (ctx *astRenameContext) collectArrowFunctionBindings(function *ast.ArrowFunctionLiteral) {
	if function == nil {
		return
	}
	scope := ctx.scopeByExactRange("arrow-function", astNodeStart(function), astNodeEnd(function))
	if scope == nil {
		scope = ctx.innermostScopeAt(astNodeStart(function))
	}
	ctx.collectParameterList(function.ParameterList, scope, sliceASTNodeSource(ctx.source, function))
}

func (ctx *astRenameContext) collectParameterList(params *ast.ParameterList, scope *astScope, functionSrc string) {
	if params == nil || scope == nil {
		return
	}
	total := len(params.List)
	if params.Rest != nil {
		total++
	}
	for i, binding := range params.List {
		ctx.collectBinding(binding, scope, "param", i, total, functionSrc)
	}
	if params.Rest != nil {
		if target, ok := params.Rest.(ast.BindingTarget); ok {
			ctx.collectBindingTarget(target, nil, scope, "param", total-1, total, functionSrc)
		} else {
			ctx.markIdentifierOffsets(params.Rest, "unsupported-rest-parameter")
		}
	}
}

func (ctx *astRenameContext) collectBindingList(list []*ast.Binding, scope *astScope, kind string, paramIndex, paramTotal int, functionSrc string) {
	for _, binding := range list {
		ctx.collectBinding(binding, scope, kind, paramIndex, paramTotal, functionSrc)
	}
}

func (ctx *astRenameContext) collectBinding(binding *ast.Binding, scope *astScope, kind string, paramIndex, paramTotal int, functionSrc string) {
	if binding == nil {
		return
	}
	ctx.collectBindingTarget(binding.Target, binding.Initializer, scope, kind, paramIndex, paramTotal, functionSrc)
}

func (ctx *astRenameContext) collectBindingTarget(target ast.BindingTarget, initializer ast.Expression, scope *astScope, kind string, paramIndex, paramTotal int, functionSrc string) {
	identifier, ok := target.(*ast.Identifier)
	if !ok || identifier == nil {
		ctx.markPatternOffsets(target, "unsupported-binding-pattern")
		return
	}
	ctx.addBinding(identifier, initializer, scope, kind, paramIndex, paramTotal, functionSrc)
}

func (ctx *astRenameContext) addBinding(identifier *ast.Identifier, initializer ast.Expression, scope *astScope, kind string, paramIndex, paramTotal int, functionSrc string) {
	if identifier == nil || scope == nil {
		return
	}
	name := identifier.Name.String()
	if name == "" {
		return
	}
	binding := &astBinding{
		name:        name,
		scope:       scope,
		offset:      astNodeStart(identifier),
		kind:        kind,
		initExpr:    initializer,
		paramIndex:  paramIndex,
		paramTotal:  paramTotal,
		functionSrc: functionSrc,
	}
	if initializer != nil {
		binding.initSource = sliceASTNodeSource(ctx.source, initializer)
	}
	scope.bindings = append(scope.bindings, binding)
	scope.used[name] = struct{}{}
	ctx.bindings = append(ctx.bindings, binding)
}

func (ctx *astRenameContext) collectSkipOffsets() {
	walkASTNode(ctx.program, func(node ast.Node) {
		switch item := node.(type) {
		case *ast.PropertyKeyed:
			if item != nil && !item.Computed {
				ctx.markIdentifierOffsets(item.Key, "object-property-key")
			}
		case *ast.PropertyShort:
			if item == nil {
				return
			}
			name := item.Name.Name.String()
			offset := astNodeStart(&item.Name)
			ctx.shorthandByName[name] = append(ctx.shorthandByName[name], offset)
			ctx.skipOffsets[offset] = "object-shorthand-property"
		case *ast.MethodDefinition:
			if item != nil && !item.Computed {
				ctx.markIdentifierOffsets(item.Key, "class-method-key")
			}
		case *ast.FieldDefinition:
			if item != nil && !item.Computed {
				ctx.markIdentifierOffsets(item.Key, "class-field-key")
			}
		case *ast.LabelledStatement:
			if item != nil && item.Label != nil {
				ctx.skipOffsets[astNodeStart(item.Label)] = "statement-label"
			}
		case *ast.BranchStatement:
			if item != nil && item.Label != nil {
				ctx.skipOffsets[astNodeStart(item.Label)] = "branch-label"
			}
		case *ast.MetaProperty:
			if item != nil {
				ctx.markIdentifierOffsets(item.Meta, "meta-property")
				ctx.markIdentifierOffsets(item.Property, "meta-property")
			}
		case *ast.ObjectPattern:
			ctx.markIdentifierOffsets(item, "object-pattern")
		case *ast.ArrayPattern:
			ctx.markIdentifierOffsets(item, "array-pattern")
		}
	})
}

func (ctx *astRenameContext) markPatternOffsets(target ast.Node, reason string) {
	if target == nil {
		return
	}
	ctx.markIdentifierOffsets(target, reason)
}

func (ctx *astRenameContext) markIdentifierOffsets(node ast.Node, reason string) {
	if node == nil {
		return
	}
	walkASTNode(node, func(inner ast.Node) {
		identifier, ok := inner.(*ast.Identifier)
		if !ok || identifier == nil {
			return
		}
		ctx.skipOffsets[astNodeStart(identifier)] = reason
	})
	if identifier, ok := node.(*ast.Identifier); ok && identifier != nil {
		ctx.skipOffsets[astNodeStart(identifier)] = reason
	}
}

func (ctx *astRenameContext) collectOccurrences() {
	seen := make(map[int]struct{})
	walkASTNode(ctx.program, func(node ast.Node) {
		identifier, ok := node.(*ast.Identifier)
		if !ok || identifier == nil {
			return
		}
		start := astNodeStart(identifier)
		end := astNodeEnd(identifier)
		if start < 0 || end > len(ctx.source) || start >= end {
			return
		}
		if _, ok := seen[start]; ok {
			return
		}
		seen[start] = struct{}{}
		if _, skip := ctx.skipOffsets[start]; skip {
			return
		}
		ctx.occurrences = append(ctx.occurrences, astIdentifierOccurrence{
			name:  identifier.Name.String(),
			start: start,
			end:   end,
		})
	})
	sort.Slice(ctx.occurrences, func(i, j int) bool {
		return ctx.occurrences[i].start < ctx.occurrences[j].start
	})
}

func (ctx *astRenameContext) resolveOccurrences() {
	for i := range ctx.occurrences {
		occ := &ctx.occurrences[i]
		scope := ctx.innermostScopeAt(occ.start)
		occ.binder = ctx.resolveBinding(scope, occ.name)
	}
}

func (ctx *astRenameContext) indexOccurrences() {
	ctx.occurrencesByBinding = make(map[*astBinding][]astIdentifierOccurrence)
	for _, occ := range ctx.occurrences {
		if occ.binder == nil {
			continue
		}
		ctx.occurrencesByBinding[occ.binder] = append(ctx.occurrencesByBinding[occ.binder], occ)
	}
}

func (ctx *astRenameContext) resolveBinding(scope *astScope, name string) *astBinding {
	for current := scope; current != nil; current = current.parent {
		for _, binding := range current.bindings {
			if binding.name == name {
				return binding
			}
		}
	}
	return nil
}

func (ctx *astRenameContext) planRenames() []astRenameCandidate {
	candidates := make([]astRenameCandidate, 0)
	usedTargets := make(map[*astScope]map[string]struct{})
	for _, scope := range ctx.scopes {
		usedTargets[scope] = scope.collectVisibleNames()
	}

	for _, binding := range ctx.bindings {
		target, reason, confidence := ctx.inferTargetName(binding)
		if target == "" || target == binding.name {
			continue
		}
		if !isValidIdentifierName(target) {
			continue
		}
		if ctx.hasShorthandHazard(binding) {
			continue
		}
		target = allocateASTBindingName(target, binding, usedTargets[binding.scope])
		if target == binding.name {
			continue
		}
		replacements := ctx.replacementsForBinding(binding, target)
		if len(replacements) == 0 {
			continue
		}
		candidates = append(candidates, astRenameCandidate{
			binding:      binding,
			target:       target,
			reason:       reason,
			confidence:   confidence,
			replacements: replacements,
		})
		usedTargets[binding.scope][target] = struct{}{}
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].binding.offset < candidates[j].binding.offset
	})
	return candidates
}

func (ctx *astRenameContext) inferTargetName(binding *astBinding) (string, string, string) {
	if binding == nil || !isASTRenameCandidate(binding.name) || isReservedIdentifier(binding.name) {
		return "", "", ""
	}
	if binding.kind == "function" {
		target := ctx.inferFunctionName(binding)
		if target != "" {
			return target, "local function call/export context", ASTConfidenceMedium
		}
		return "", "", ""
	}
	if target := ctx.inferRequireAlias(binding); target != "" {
		return target, "require alias inferred from module path", ASTConfidenceHigh
	}
	if binding.kind == "param" {
		if binding.paramTotal == 2 && binding.paramIndex == 0 && strings.Contains(binding.functionSrc, "new Promise") {
			return "resolve", "Promise executor parameter", ASTConfidenceHigh
		}
		if binding.paramTotal == 2 && binding.paramIndex == 1 && strings.Contains(binding.functionSrc, "new Promise") {
			return "reject", "Promise executor parameter", ASTConfidenceHigh
		}
		if strings.Contains(binding.functionSrc, "controllerName") && containsMethodNameLiteral(binding.functionSrc) {
			return "params", "API wrapper parameter", ASTConfidenceHigh
		}
		if strings.Contains(binding.functionSrc, "wx.request") || strings.Contains(binding.functionSrc, "uni.request") ||
			strings.Contains(binding.functionSrc, ".request({") || strings.Contains(binding.functionSrc, binding.name+".url") ||
			strings.Contains(binding.functionSrc, binding.name+".method") || strings.Contains(binding.functionSrc, binding.name+".data") {
			return "options", "request wrapper parameter", ASTConfidenceHigh
		}
		if ctx.bindingHasMember(binding, "detail", "currentTarget", "target", "preventDefault", "stopPropagation") {
			return "event", "page event parameter", ASTConfidenceMedium
		}
		if ctx.bindingHasMember(binding, "statusCode", "headers", "data", "errMsg") {
			return "response", "request response parameter", ASTConfidenceMedium
		}
		if ctx.bindingHasMember(binding, "userId", "openid", "phone", "mobile") {
			return "params", "business parameter object", ASTConfidenceMedium
		}
		return "", "", ""
	}

	init := binding.initSource
	switch {
	case strings.Contains(init, "getApp()"):
		return "app", "getApp() local alias", ASTConfidenceHigh
	case strings.HasPrefix(strings.TrimSpace(init), "{") && strings.Contains(init, "controllerName") && containsMethodNameLiteral(init):
		return "requestData", "API request payload object", ASTConfidenceHigh
	case strings.Contains(strings.ToLower(init), "token"):
		return "token", "token initializer", ASTConfidenceLow
	}

	usage := ctx.bindingUsageSource(binding)
	lowerUsage := strings.ToLower(usage)
	switch {
	case ctx.bindingHasMember(binding, "globalData"):
		return "app", "direct globalData owner usage", ASTConfidenceMedium
	case strings.Contains(lowerUsage, "token"):
		return "token", "token usage", ASTConfidenceLow
	case strings.Contains(lowerUsage, "route") || strings.Contains(lowerUsage, "path"):
		return "route", "route/path usage", ASTConfidenceLow
	case strings.Contains(lowerUsage, "query"):
		return "query", "query usage", ASTConfidenceLow
	case strings.Contains(usage, ".data") && strings.Contains(usage, ".statusCode"):
		return "apiResponse", "API response usage", ASTConfidenceMedium
	case strings.Contains(lowerUsage, "config") || strings.Contains(usage, ".baseApiUrl") || strings.Contains(usage, ".baseImgUrl"):
		return "config", "config usage", ASTConfidenceLow
	}
	return "", "", ""
}

func (ctx *astRenameContext) inferRequireAlias(binding *astBinding) string {
	literal := requireLiteralFromExpression(binding.initExpr)
	if literal == "" {
		return ""
	}
	resolved := resolveRequirePath(ctx.relPath, literal, nil)
	base := strings.TrimSuffix(path.Base(resolved), ".js")
	dir := path.Base(path.Dir(resolved))
	baseLower := strings.ToLower(base)
	switch {
	case strings.Contains(baseLower, "request"):
		return "requestClient"
	case strings.Contains(baseLower, "crypto") || strings.Contains(baseLower, "encrypt") || strings.Contains(baseLower, "sm2") || strings.Contains(baseLower, "sm3") || strings.Contains(baseLower, "sm4"):
		return "cryptoEncrypt"
	case strings.Contains(baseLower, "config"):
		return "config"
	case strings.HasPrefix(resolved, "api/") || dir == "api":
		return "api" + exportNamePart(base)
	case strings.Contains(baseLower, "util"):
		return "utils"
	default:
		return lowerCamelFromPath(base)
	}
}

func (ctx *astRenameContext) inferFunctionName(binding *astBinding) string {
	usage := ctx.bindingUsageSource(binding)
	if strings.Contains(usage, ".request(") || strings.Contains(binding.functionSrc, "wx.request") {
		return "request"
	}
	if strings.Contains(usage, ".then(") || strings.Contains(usage, "Promise") {
		return "handleResponse"
	}
	return ""
}

func (ctx *astRenameContext) bindingUsageContains(binding *astBinding, fragments []string) bool {
	usage := ctx.bindingUsageSource(binding)
	for _, fragment := range fragments {
		if strings.Contains(usage, fragment) {
			return true
		}
	}
	return false
}

func (ctx *astRenameContext) bindingHasMember(binding *astBinding, members ...string) bool {
	if binding == nil {
		return false
	}
	memberSet := make(map[string]struct{}, len(members))
	for _, member := range members {
		memberSet[member] = struct{}{}
	}
	for _, occ := range ctx.occurrencesByBinding[binding] {
		if occ.start == binding.offset {
			continue
		}
		if occ.end >= len(ctx.source) || ctx.source[occ.end] != '.' {
			continue
		}
		memberStart := occ.end + 1
		memberEnd := memberStart
		for memberEnd < len(ctx.source) {
			ch := rune(ctx.source[memberEnd])
			if ch != '$' && ch != '_' && !unicode.IsLetter(ch) && !unicode.IsDigit(ch) {
				break
			}
			memberEnd++
		}
		if _, ok := memberSet[ctx.source[memberStart:memberEnd]]; ok {
			return true
		}
	}
	return false
}

func (ctx *astRenameContext) bindingUsageSource(binding *astBinding) string {
	if binding == nil {
		return ""
	}
	parts := make([]string, 0)
	for _, occ := range ctx.occurrencesByBinding[binding] {
		if occ.start == binding.offset {
			continue
		}
		start := max(occ.start-48, 0)
		end := min(occ.end+96, len(ctx.source))
		parts = append(parts, ctx.source[start:end])
	}
	return strings.Join(parts, "\n")
}

func (ctx *astRenameContext) replacementsForBinding(binding *astBinding, target string) []astReplacement {
	replacements := make([]astReplacement, 0)
	for _, occ := range ctx.occurrencesByBinding[binding] {
		if ctx.source[occ.start:occ.end] != binding.name {
			continue
		}
		replacements = append(replacements, astReplacement{
			start: occ.start,
			end:   occ.end,
			from:  binding.name,
			to:    target,
		})
	}
	return replacements
}

func (ctx *astRenameContext) hasShorthandHazard(binding *astBinding) bool {
	if binding == nil {
		return false
	}
	for _, offset := range ctx.shorthandByName[binding.name] {
		scope := ctx.innermostScopeAt(offset)
		if ctx.resolveBinding(scope, binding.name) == binding {
			return true
		}
	}
	return false
}

func (ctx *astRenameContext) skippedItems() []ASTSkipItem {
	items := make([]ASTSkipItem, 0)
	for _, binding := range ctx.bindings {
		if !isASTRenameCandidate(binding.name) || isReservedIdentifier(binding.name) {
			continue
		}
		if ctx.hasShorthandHazard(binding) {
			items = append(items, ASTSkipItem{
				Name:       binding.name,
				ScopeKind:  binding.scope.kind,
				Reason:     "object-shorthand-property",
				LineNumber: lineNumberAtOffset(ctx.source, binding.offset),
			})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].LineNumber != items[j].LineNumber {
			return items[i].LineNumber < items[j].LineNumber
		}
		return items[i].Name < items[j].Name
	})
	return items
}

func (scope *astScope) collectVisibleNames() map[string]struct{} {
	used := make(map[string]struct{})
	for current := scope; current != nil; current = current.parent {
		for _, binding := range current.bindings {
			used[binding.name] = struct{}{}
		}
	}
	return used
}

func allocateASTBindingName(candidate string, binding *astBinding, used map[string]struct{}) string {
	if used == nil {
		return candidate
	}
	if _, exists := used[candidate]; !exists {
		return candidate
	}
	if candidate == binding.name {
		return candidate
	}
	for i := 2; ; i++ {
		next := fmt.Sprintf("%s%d", candidate, i)
		if _, exists := used[next]; !exists {
			return next
		}
	}
}

func (ctx *astRenameContext) innermostScopeAt(offset int) *astScope {
	var best *astScope
	for _, scope := range ctx.scopes {
		if scope.start <= offset && offset < scope.end {
			if best == nil || scopeRange(scope) < scopeRange(best) ||
				(scopeRange(scope) == scopeRange(best) && best.kind == "program" && scope.kind != "program") {
				best = scope
			}
		}
	}
	if best == nil && len(ctx.scopes) > 0 {
		return ctx.scopes[0]
	}
	return best
}

func (ctx *astRenameContext) functionScopeAt(offset int) *astScope {
	var best *astScope
	for _, scope := range ctx.scopes {
		if scope.kind != "function" && scope.kind != "arrow-function" && scope.kind != "program" {
			continue
		}
		if scope.start <= offset && offset < scope.end {
			if best == nil || scopeRange(scope) < scopeRange(best) ||
				(scopeRange(scope) == scopeRange(best) && best.kind == "program" && scope.kind != "program") {
				best = scope
			}
		}
	}
	if best != nil {
		return best
	}
	if len(ctx.scopes) > 0 {
		return ctx.scopes[0]
	}
	return nil
}

func (ctx *astRenameContext) scopeByExactRange(kind string, start, end int) *astScope {
	for _, scope := range ctx.scopes {
		if scope.kind == kind && scope.start == start && scope.end == end {
			return scope
		}
	}
	return nil
}

func applyASTReplacements(source string, replacements []astReplacement) (string, bool) {
	if len(replacements) == 0 {
		return source, false
	}
	sort.Slice(replacements, func(i, j int) bool {
		return replacements[i].start > replacements[j].start
	})
	result := source
	changed := false
	seen := make(map[int]struct{}, len(replacements))
	for _, replacement := range replacements {
		if replacement.start < 0 || replacement.end > len(result) || replacement.start >= replacement.end {
			continue
		}
		if _, ok := seen[replacement.start]; ok {
			continue
		}
		seen[replacement.start] = struct{}{}
		if result[replacement.start:replacement.end] != replacement.from {
			continue
		}
		result = result[:replacement.start] + replacement.to + result[replacement.end:]
		changed = true
	}
	return result, changed
}

func backupPreASTSource(rootDir, rel, source string) error {
	backupPath := filepath.Join(rootDir, reportDirName, preASTSourcesDirName, filepath.FromSlash(rel))
	if _, err := os.Stat(backupPath); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(backupPath, []byte(source), 0644)
}

// ASTRollbackReport 描述 AST 回滚结果。
type ASTRollbackReport struct {
	RestoredFiles []string `json:"restored_files"`
}

// RollbackASTRenames 从 .gwxapkg/pre_ast_sources 恢复 AST 写回前的文件。
func RollbackASTRenames(rootDir string) (*ASTRollbackReport, error) {
	rootAbs, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("解析输出目录失败: %w", err)
	}
	backupRoot := filepath.Join(rootAbs, reportDirName, preASTSourcesDirName)
	if _, err := os.Stat(backupRoot); err != nil {
		return nil, fmt.Errorf("未找到 AST 回滚目录: %w", err)
	}
	report := &ASTRollbackReport{}
	err = filepath.WalkDir(backupRoot, func(filePath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(backupRoot, filePath)
		if err != nil {
			return err
		}
		target := filepath.Join(rootAbs, rel)
		data, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0644); err != nil {
			return err
		}
		report.RestoredFiles = append(report.RestoredFiles, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(report.RestoredFiles)
	return report, nil
}

func writeASTRenameDiff(rootDir string, changes []astFileChange) error {
	reportDir := filepath.Join(rootDir, reportDirName)
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return err
	}
	var builder strings.Builder
	builder.WriteString("# AST 重命名 Diff\n\n")
	if len(changes) == 0 {
		builder.WriteString("本次没有 AST 写回。\n")
		return os.WriteFile(filepath.Join(reportDir, astRenameDiffFileName), []byte(builder.String()), 0644)
	}
	for _, change := range changes {
		builder.WriteString("## ")
		builder.WriteString("`")
		builder.WriteString(change.relPath)
		builder.WriteString("`\n\n")
		for _, rename := range change.renames {
			if !rename.Applied {
				continue
			}
			builder.WriteString(fmt.Sprintf("- `%s` -> `%s` | `%s` | line `%d` | replacements `%d`\n",
				rename.BeforeName, rename.AfterName, rename.Confidence, rename.LineNumber, rename.ReplacementCount))
			if rename.SourceSnippet != "" {
				builder.WriteString("\n```js\n")
				builder.WriteString(rename.SourceSnippet)
				builder.WriteString("\n```\n")
			}
		}
		builder.WriteString("\n")
	}
	return os.WriteFile(filepath.Join(reportDir, astRenameDiffFileName), []byte(builder.String()), 0644)
}

func writeASTRenamePatch(rootDir string, changes []astFileChange) error {
	reportDir := filepath.Join(rootDir, reportDirName)
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return err
	}
	var builder strings.Builder
	for _, change := range changes {
		builder.WriteString(buildWholeFilePatch(change.relPath, change.before, change.after))
	}
	return os.WriteFile(filepath.Join(reportDir, astRenamePatchFileName), []byte(builder.String()), 0644)
}

func buildWholeFilePatch(rel, before, after string) string {
	beforeLines := splitPatchLines(before)
	afterLines := splitPatchLines(after)
	var builder strings.Builder
	builder.WriteString("--- a/")
	builder.WriteString(rel)
	builder.WriteString("\n+++ b/")
	builder.WriteString(rel)
	builder.WriteString("\n")
	builder.WriteString(fmt.Sprintf("@@ -1,%d +1,%d @@\n", len(beforeLines), len(afterLines)))
	for _, line := range beforeLines {
		builder.WriteString("-")
		builder.WriteString(line)
	}
	for _, line := range afterLines {
		builder.WriteString("+")
		builder.WriteString(line)
	}
	return builder.String()
}

func splitPatchLines(text string) []string {
	if text == "" {
		return []string{""}
	}
	lines := strings.SplitAfter(text, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return []string{""}
	}
	for i, line := range lines {
		if !strings.HasSuffix(line, "\n") {
			lines[i] = line + "\n"
		}
	}
	return lines
}

func astRenameSnippet(source string, offset int) string {
	if offset < 0 {
		return ""
	}
	start := offset
	for start > 0 && source[start-1] != '\n' {
		start--
	}
	for i := 0; i < 2 && start > 0; i++ {
		start--
		for start > 0 && source[start-1] != '\n' {
			start--
		}
	}
	end := offset
	for end < len(source) && source[end] != '\n' {
		end++
	}
	for i := 0; i < 2 && end < len(source); i++ {
		end++
		for end < len(source) && source[end] != '\n' {
			end++
		}
	}
	return strings.TrimSpace(source[start:end])
}

func requireLiteralFromExpression(expr ast.Expression) string {
	call, ok := expr.(*ast.CallExpression)
	if !ok || call == nil {
		return ""
	}
	callee, ok := call.Callee.(*ast.Identifier)
	if !ok || callee == nil || callee.Name.String() != "require" {
		return ""
	}
	if len(call.ArgumentList) != 1 {
		return ""
	}
	literal, ok := call.ArgumentList[0].(*ast.StringLiteral)
	if !ok || literal == nil {
		return ""
	}
	return literal.Value.String()
}

func isASTRenameCandidate(name string) bool {
	if name == "" {
		return false
	}
	return isShortIdentifier(name) || hexIdentifierPattern.MatchString(name)
}

func isReservedIdentifier(name string) bool {
	switch name {
	case "wx", "uni", "getApp", "require", "module", "exports", "Page", "App", "Component",
		"Promise", "Object", "Array", "String", "Number", "Boolean", "JSON", "Math", "Date",
		"console", "window", "document", "frames", "self", "location", "global", "globalThis",
		"undefined", "arguments", "this":
		return true
	default:
		return false
	}
}

func containsMethodNameLiteral(source string) bool {
	return strings.Contains(source, "methodsName") || strings.Contains(source, "methodName")
}

func isValidIdentifierName(name string) bool {
	if name == "" || isReservedIdentifier(name) {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if r != '$' && r != '_' && !unicode.IsLetter(r) {
				return false
			}
			continue
		}
		if r != '$' && r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func lowerCamelFromPath(value string) string {
	value = strings.Trim(value, "_-. ")
	if value == "" {
		return ""
	}
	parts := regexp.MustCompile(`[^A-Za-z0-9]+`).Split(value, -1)
	result := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		lower := strings.ToLower(part)
		if result == "" {
			result = lower
			continue
		}
		result += exportNamePart(lower)
	}
	return result
}

func exportNamePart(value string) string {
	value = lowerCamelFromPath(value)
	if value == "" {
		return ""
	}
	runes := []rune(value)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func sliceASTNodeSource(source string, node ast.Node) string {
	if node == nil {
		return ""
	}
	start := astNodeStart(node)
	end := astNodeEnd(node)
	if start < 0 || end > len(source) || start >= end {
		return ""
	}
	return source[start:end]
}

func astNodeStart(node ast.Node) int {
	if node == nil {
		return -1
	}
	return max(int(node.Idx0())-1, 0)
}

func astNodeEnd(node ast.Node) int {
	if node == nil {
		return -1
	}
	return max(int(node.Idx1())-1, 0)
}

func walkASTNode(node ast.Node, fn func(ast.Node)) {
	if isNilASTNode(node) {
		return
	}
	fn(node)
	walkASTStructFields(reflect.ValueOf(node), fn)
}

func isNilASTNode(node ast.Node) bool {
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

func walkASTStructFields(value reflect.Value, fn func(ast.Node)) {
	if !value.IsValid() {
		return
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return
		}
		if value.Elem().Kind() == reflect.Struct && value.Elem().Type().PkgPath() != gojaASTPackagePath {
			return
		}
		walkASTStructFields(value.Elem(), fn)
		return
	}

	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return
		}
		elem := value.Elem()
		if elem.CanInterface() {
			if node, ok := elem.Interface().(ast.Node); ok {
				walkASTNode(node, fn)
				return
			}
		}
		if elem.Kind() == reflect.Struct && elem.Type().PkgPath() != gojaASTPackagePath {
			return
		}
		walkASTStructFields(elem, fn)
	case reflect.Struct:
		if value.Type().PkgPath() != "" && value.Type().PkgPath() != gojaASTPackagePath {
			return
		}
		for i := 0; i < value.NumField(); i++ {
			field := value.Field(i)
			if !field.CanInterface() {
				continue
			}
			if node, ok := field.Interface().(ast.Node); ok {
				walkASTNode(node, fn)
				continue
			}
			walkASTStructFields(field, fn)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			item := value.Index(i)
			if item.CanInterface() {
				if node, ok := item.Interface().(ast.Node); ok {
					walkASTNode(node, fn)
					continue
				}
			}
			walkASTStructFields(item, fn)
		}
	}
}

func writeASTRenameReport(rootDir string, report *ASTRenameReport) error {
	reportDir := filepath.Join(rootDir, reportDirName)
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(reportDir, astRenameReportFileName), data, 0644)
}
