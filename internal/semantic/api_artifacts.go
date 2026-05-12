package semantic

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	apiCallChainJSONFileName = "api_call_chain.json"
	apiCallChainMDFileName   = "api_call_chain.md"
	apiPseudoMDFileName      = "api_pseudo.md"
	pseudoAPIDirName         = "pseudo_api"
)

type jsFunctionRange struct {
	Name  string
	Kind  string
	Start int
	End   int
	Line  int
}

var (
	namedFunctionPattern = regexp.MustCompile(`(?m)\bfunction\s+([A-Za-z_$][\w$]*)\s*\([^)]*\)\s*\{`)
	varFunctionPattern   = regexp.MustCompile(`(?m)\b(?:var|let|const)\s+([A-Za-z_$][\w$]*)\s*=\s*function\s*\([^)]*\)\s*\{`)
	propFunctionPattern  = regexp.MustCompile(`(?m)\b([A-Za-z_$][\w$]*)\s*:\s*function\s*\([^)]*\)\s*\{`)
	arrowFunctionPattern = regexp.MustCompile(`(?m)\b(?:var|let|const)\s+([A-Za-z_$][\w$]*)\s*=\s*(?:\([^)]*\)|[A-Za-z_$][\w$]*)\s*=>\s*\{`)
)

// BuildAPICallChains 根据 api_map 的调用点生成页面到后端方法的静态调用链。
func BuildAPICallChains(rootDir string, report *APIMapReport) []APICallChain {
	if report == nil {
		return nil
	}
	results := make([]APICallChain, 0)
	for _, endpoint := range report.Endpoints {
		for _, call := range endpoint.CallSites {
			chain := buildAPICallChain(rootDir, endpoint, call)
			results = append(results, chain)
		}
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].PageFile != results[j].PageFile {
			return results[i].PageFile < results[j].PageFile
		}
		if results[i].LineNumber != results[j].LineNumber {
			return results[i].LineNumber < results[j].LineNumber
		}
		return results[i].APIFunction < results[j].APIFunction
	})
	return results
}

func buildAPICallChain(rootDir string, endpoint APIEndpointEntry, call APICallSite) APICallChain {
	chain := APICallChain{
		PageRoute:      pageRouteFromJS(call.FilePath),
		PageFile:       call.FilePath,
		APIFile:        endpoint.FilePath,
		APIFunction:    endpoint.FunctionName,
		ControllerName: endpoint.ControllerName,
		MethodsName:    endpoint.MethodsName,
		HTTPMethod:     endpoint.HTTPMethod,
		ParamFields:    endpoint.ParamFields,
		LineNumber:     call.LineNumber,
		Expression:     call.Expression,
		Confidence:     ASTConfidenceHigh,
	}

	data, err := os.ReadFile(filepath.Join(rootDir, filepath.FromSlash(call.FilePath)))
	if err != nil {
		chain.Confidence = ASTConfidenceLow
		return chain
	}
	text := string(data)
	offset := offsetForLine(text, call.LineNumber)
	functions := collectJSFunctionRanges(text)
	current := innermostFunctionAt(functions, offset)
	if current != nil {
		steps := traceLocalAPICallers(text, functions, current, 4)
		chain.Steps = append(chain.Steps, steps...)
		chain.Steps = append(chain.Steps, APICallChainStep{
			FilePath:     call.FilePath,
			FunctionName: current.Name,
			Kind:         current.Kind,
			LineNumber:   current.Line,
		})
		chain.EntryFunction = chain.Steps[0].FunctionName
	}
	chain.Steps = append(chain.Steps, APICallChainStep{
		FilePath:     endpoint.FilePath,
		FunctionName: endpoint.FunctionName,
		Kind:         "api-call",
		LineNumber:   call.LineNumber,
	})
	return chain
}

func collectJSFunctionRanges(text string) []jsFunctionRange {
	results := make([]jsFunctionRange, 0)
	addMatches := func(pattern *regexp.Regexp, kind string) {
		for _, match := range pattern.FindAllStringSubmatchIndex(text, -1) {
			if len(match) < 4 {
				continue
			}
			open := strings.LastIndex(text[match[0]:match[1]], "{")
			if open < 0 {
				continue
			}
			open += match[0]
			end := findMatchingBrace(text, open)
			if end <= open {
				continue
			}
			results = append(results, jsFunctionRange{
				Name:  text[match[2]:match[3]],
				Kind:  kind,
				Start: match[0],
				End:   end + 1,
				Line:  lineNumberAtOffset(text, match[0]),
			})
		}
	}
	addMatches(namedFunctionPattern, "function")
	addMatches(varFunctionPattern, "local-helper")
	addMatches(propFunctionPattern, "object-method")
	addMatches(arrowFunctionPattern, "arrow-helper")
	sort.Slice(results, func(i, j int) bool {
		if results[i].Start != results[j].Start {
			return results[i].Start < results[j].Start
		}
		return results[i].End < results[j].End
	})
	return results
}

func innermostFunctionAt(functions []jsFunctionRange, offset int) *jsFunctionRange {
	var best *jsFunctionRange
	for i := range functions {
		fn := &functions[i]
		if fn.Start <= offset && offset < fn.End {
			if best == nil || (fn.End-fn.Start) < (best.End-best.Start) {
				best = fn
			}
		}
	}
	return best
}

func traceLocalAPICallers(text string, functions []jsFunctionRange, current *jsFunctionRange, maxDepth int) []APICallChainStep {
	if current == nil || maxDepth <= 0 {
		return nil
	}
	steps := make([]APICallChainStep, 0)
	seen := map[string]bool{current.Name: true}
	target := current
	for depth := 0; depth < maxDepth; depth++ {
		caller := findLocalCaller(text, functions, target.Name, seen)
		if caller == nil {
			break
		}
		steps = append([]APICallChainStep{{
			FunctionName: caller.Name,
			Kind:         caller.Kind,
			LineNumber:   caller.Line,
		}}, steps...)
		seen[caller.Name] = true
		target = caller
	}
	return steps
}

func findLocalCaller(text string, functions []jsFunctionRange, callee string, seen map[string]bool) *jsFunctionRange {
	if callee == "" {
		return nil
	}
	pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(callee) + `\s*\(`)
	for i := range functions {
		fn := &functions[i]
		if seen[fn.Name] {
			continue
		}
		body := text[fn.Start:fn.End]
		if pattern.FindStringIndex(body) != nil {
			return fn
		}
	}
	return nil
}

func pageRouteFromJS(rel string) string {
	if strings.HasSuffix(rel, ".js") {
		return strings.TrimSuffix(rel, ".js")
	}
	return rel
}

func offsetForLine(text string, line int) int {
	if line <= 1 {
		return 0
	}
	current := 1
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			current++
			if current == line {
				return i + 1
			}
		}
	}
	return len(text)
}

func writeAPICallChain(rootDir string, chains []APICallChain) error {
	reportDir := filepath.Join(rootDir, reportDirName)
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(chains, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(reportDir, apiCallChainJSONFileName), data, 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(reportDir, apiCallChainMDFileName), []byte(buildAPICallChainMarkdown(chains)), 0644)
}

func buildAPICallChainMarkdown(chains []APICallChain) string {
	var builder strings.Builder
	builder.WriteString("# API 调用链\n\n")
	if len(chains) == 0 {
		builder.WriteString("未识别到页面 API 调用链。\n")
		return builder.String()
	}
	for _, chain := range chains {
		builder.WriteString(fmt.Sprintf("- `%s` line `%d` -> `%s.%s` via `%s`\n",
			chain.PageFile, chain.LineNumber, chain.ControllerName, chain.MethodsName, chain.APIFunction))
		if len(chain.Steps) > 0 {
			for _, step := range chain.Steps {
				file := step.FilePath
				if file == "" {
					file = chain.PageFile
				}
				builder.WriteString(fmt.Sprintf("  - `%s:%d` `%s` (%s)\n", file, step.LineNumber, step.FunctionName, step.Kind))
			}
		}
	}
	return builder.String()
}

func writeAPIPseudo(rootDir string, report *APIMapReport) error {
	if report == nil {
		return nil
	}
	reportDir := filepath.Join(rootDir, reportDirName)
	pseudoDir := filepath.Join(reportDir, pseudoAPIDirName)
	if err := os.MkdirAll(pseudoDir, 0755); err != nil {
		return err
	}
	groups := make(map[string][]APIEndpointEntry)
	for _, endpoint := range report.Endpoints {
		controller := endpoint.ControllerName
		if controller == "" {
			controller = "unknown"
		}
		groups[controller] = append(groups[controller], endpoint)
	}
	controllers := make([]string, 0, len(groups))
	for controller := range groups {
		controllers = append(controllers, controller)
	}
	sort.Strings(controllers)
	var md strings.Builder
	md.WriteString("# API 伪代码\n\n")
	for _, controller := range controllers {
		endpoints := groups[controller]
		sort.Slice(endpoints, func(i, j int) bool {
			return endpoints[i].FunctionName < endpoints[j].FunctionName
		})
		content := buildPseudoAPIFile(controller, endpoints)
		fileName := controllerSlug(controller) + ".js"
		if err := os.WriteFile(filepath.Join(pseudoDir, fileName), []byte(content), 0644); err != nil {
			return err
		}
		md.WriteString(fmt.Sprintf("## `%s`\n\n```js\n%s\n```\n\n", path.Join(reportDirName, pseudoAPIDirName, fileName), strings.TrimSpace(content)))
	}
	return os.WriteFile(filepath.Join(reportDir, apiPseudoMDFileName), []byte(md.String()), 0644)
}

func buildPseudoAPIFile(controller string, endpoints []APIEndpointEntry) string {
	var builder strings.Builder
	builder.WriteString("// 由 Gwxapkg 生成的审计伪代码，不参与运行。\n\n")
	for _, endpoint := range endpoints {
		params := validParamFields(endpoint.ParamFields)
		builder.WriteString("function ")
		builder.WriteString(endpoint.FunctionName)
		builder.WriteString("({ ")
		builder.WriteString(strings.Join(params, ", "))
		builder.WriteString(" }) {\n")
		builder.WriteString("  return requestAPI({\n")
		builder.WriteString(fmt.Sprintf("    httpMethod: %q,\n", emptyAsDefault(endpoint.HTTPMethod, "GET")))
		builder.WriteString(fmt.Sprintf("    controllerName: %q,\n", controller))
		builder.WriteString(fmt.Sprintf("    methodsName: %q,\n", endpoint.MethodsName))
		builder.WriteString("    data: { ")
		builder.WriteString(strings.Join(params, ", "))
		builder.WriteString(" }\n")
		builder.WriteString("  });\n")
		builder.WriteString("}\n\n")
	}
	return builder.String()
}

func validParamFields(fields []string) []string {
	results := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if isValidIdentifierName(field) {
			results = append(results, field)
		}
	}
	if len(results) == 0 {
		return []string{"params"}
	}
	return results
}

func emptyAsDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
