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
	"time"
)

var (
	httpMethodPattern     = regexp.MustCompile(`\bmethod\s*:\s*["']([^"']+)["']`)
	urlLiteralPattern     = regexp.MustCompile(`\burl\s*:\s*["']([^"']*)["']`)
	objectKeyPattern      = regexp.MustCompile(`(?:^|[,{]\s*)([A-Za-z_$][\w$]*)\s*:`)
	requireAliasPattern   = regexp.MustCompile(`(?m)(?:\b(?:var|let|const)\s+|,\s*)?([A-Za-z_$][\w$]*)\s*=\s*require\s*\(\s*["']([^"']+\.js)["']\s*\)`)
	exportReexportPattern = regexp.MustCompile(`\bexports\.([A-Za-z_$][\w$]*)\s*=\s*([A-Za-z_$][\w$]*)\.([A-Za-z_$][\w$]*)\s*;?`)
)

// APIMapReport 是面向审计输出的接口地图。
type APIMapReport struct {
	GeneratedAt      string             `json:"generated_at"`
	SplitModuleCount int                `json:"split_module_count"`
	Endpoints        []APIEndpointEntry `json:"endpoints"`
	SplitModules     []APISplitModule   `json:"split_modules,omitempty"`
	CallChains       []APICallChain     `json:"call_chains,omitempty"`
}

// APIEndpointEntry 描述一个导出函数最终对应的后端接口。
type APIEndpointEntry struct {
	FunctionName     string        `json:"function_name"`
	ControllerName   string        `json:"controller_name"`
	MethodsName      string        `json:"methods_name"`
	HTTPMethod       string        `json:"http_method,omitempty"`
	URL              string        `json:"url,omitempty"`
	FilePath         string        `json:"file_path"`
	OriginalFilePath string        `json:"original_file_path,omitempty"`
	ParamFields      []string      `json:"param_fields,omitempty"`
	CallSites        []APICallSite `json:"call_sites,omitempty"`
}

// APICallSite 描述页面或模块中的调用点。
type APICallSite struct {
	FilePath   string `json:"file_path"`
	LineNumber int    `json:"line_number"`
	Expression string `json:"expression"`
}

// APISplitModule 描述一次 API 模块细拆。
type APISplitModule struct {
	OriginalPath string   `json:"original_path"`
	BarrelPath   string   `json:"barrel_path"`
	SplitPaths   []string `json:"split_paths"`
}

// APICallChain 描述页面到后端 controller/method 的静态调用链。
type APICallChain struct {
	PageRoute      string             `json:"page_route,omitempty"`
	PageFile       string             `json:"page_file"`
	EntryFunction  string             `json:"entry_function,omitempty"`
	APIFile        string             `json:"api_file"`
	APIFunction    string             `json:"api_function"`
	ControllerName string             `json:"controller_name"`
	MethodsName    string             `json:"methods_name"`
	HTTPMethod     string             `json:"http_method,omitempty"`
	ParamFields    []string           `json:"param_fields,omitempty"`
	LineNumber     int                `json:"line_number"`
	Expression     string             `json:"expression"`
	Confidence     string             `json:"confidence"`
	Steps          []APICallChainStep `json:"steps,omitempty"`
}

// APICallChainStep 表示一次本地 helper / lifecycle / API call 步骤。
type APICallChainStep struct {
	FilePath     string `json:"file_path"`
	FunctionName string `json:"function_name"`
	Kind         string `json:"kind"`
	LineNumber   int    `json:"line_number,omitempty"`
}

type apiFunction struct {
	FunctionName     string
	ControllerName   string
	MethodsName      string
	HTTPMethod       string
	URL              string
	FilePath         string
	OriginalFilePath string
	ParamFields      []string
	Block            string
	StartOffset      int
	EndOffset        int
	StartLine        int
	EndLine          int
}

type apiModule struct {
	RelPath   string
	FullPath  string
	Content   string
	Prefix    string
	Functions []apiFunction
}

type reexportTarget struct {
	ModulePath   string
	FunctionName string
}

// BuildAPIMap 细拆混合 API 模块，并输出 api_map.json / api_map.md。
func BuildAPIMap(rootDir string, jsFiles []string) (*APIMapReport, error) {
	modules, err := collectAPIModules(rootDir, jsFiles)
	if err != nil {
		return nil, err
	}

	report := &APIMapReport{
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		Endpoints:   make([]APIEndpointEntry, 0),
	}

	endpoints := make([]apiFunction, 0)
	for _, module := range modules {
		if shouldSplitAPIModule(module) {
			split, rewrittenFunctions, err := splitAPIModule(rootDir, module)
			if err != nil {
				return nil, err
			}
			report.SplitModules = append(report.SplitModules, split)
			endpoints = append(endpoints, rewrittenFunctions...)
			continue
		}
		endpoints = append(endpoints, module.Functions...)
	}

	allJSFiles, err := collectAllJSFiles(rootDir)
	if err != nil {
		return nil, err
	}
	reexports, err := collectReexports(rootDir, allJSFiles)
	if err != nil {
		return nil, err
	}
	report.SplitModules = mergeSplitModules(report.SplitModules, inferExistingSplitModules(reexports))
	report.SplitModuleCount = len(report.SplitModules)

	callSites, err := collectAPICallSites(rootDir, allJSFiles, endpoints, reexports)
	if err != nil {
		return nil, err
	}

	for _, endpoint := range endpoints {
		key := endpointKey(endpoint.FilePath, endpoint.FunctionName)
		entry := APIEndpointEntry{
			FunctionName:     endpoint.FunctionName,
			ControllerName:   endpoint.ControllerName,
			MethodsName:      endpoint.MethodsName,
			HTTPMethod:       endpoint.HTTPMethod,
			URL:              endpoint.URL,
			FilePath:         endpoint.FilePath,
			OriginalFilePath: endpoint.OriginalFilePath,
			ParamFields:      endpoint.ParamFields,
			CallSites:        callSites[key],
		}
		report.Endpoints = append(report.Endpoints, entry)
	}

	sortAPIMap(report)
	report.CallChains = BuildAPICallChains(rootDir, report)
	if err := writeAPIMap(rootDir, report); err != nil {
		return nil, err
	}
	return report, nil
}

func inferExistingSplitModules(reexports map[string]map[string]reexportTarget) []APISplitModule {
	results := make([]APISplitModule, 0)
	for barrelPath, exported := range reexports {
		targetSet := make(map[string]struct{})
		for _, target := range exported {
			if strings.HasPrefix(target.ModulePath, "api/") || strings.Contains(target.ModulePath, "/api/") {
				targetSet[target.ModulePath] = struct{}{}
			}
		}
		if len(targetSet) == 0 {
			continue
		}
		targets := make([]string, 0, len(targetSet))
		for target := range targetSet {
			targets = append(targets, target)
		}
		sort.Strings(targets)
		results = append(results, APISplitModule{
			OriginalPath: barrelPath,
			BarrelPath:   barrelPath,
			SplitPaths:   targets,
		})
	}
	return results
}

func mergeSplitModules(left, right []APISplitModule) []APISplitModule {
	byPath := make(map[string]APISplitModule, len(left)+len(right))
	for _, item := range append(left, right...) {
		if item.OriginalPath == "" {
			continue
		}
		item.SplitPaths = dedupeAndSort(item.SplitPaths)
		byPath[item.OriginalPath] = item
	}
	results := make([]APISplitModule, 0, len(byPath))
	for _, item := range byPath {
		results = append(results, item)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].OriginalPath < results[j].OriginalPath
	})
	return results
}

func collectAPIModules(rootDir string, jsFiles []string) ([]apiModule, error) {
	modules := make([]apiModule, 0)
	for _, rel := range jsFiles {
		if strings.HasPrefix(rel, reportDirName+"/") {
			continue
		}
		fullPath := filepath.Join(rootDir, filepath.FromSlash(rel))
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, err
		}
		functions := extractAPIFunctions(rel, string(data))
		if len(functions) == 0 {
			continue
		}
		prefix := ""
		if functions[0].StartOffset > 0 {
			prefix = strings.TrimSpace(string(data[:functions[0].StartOffset]))
		}
		modules = append(modules, apiModule{
			RelPath:   rel,
			FullPath:  fullPath,
			Content:   string(data),
			Prefix:    prefix,
			Functions: functions,
		})
	}
	return modules, nil
}

func collectAllJSFiles(rootDir string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.WalkDir(rootDir, func(filePath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if shouldSkipDir(rootDir, filePath) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.ToLower(filepath.Ext(entry.Name())) != ".js" {
			return nil
		}
		rel, err := filepath.Rel(rootDir, filePath)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func extractAPIFunctions(relPath, content string) []apiFunction {
	matches := exportFunctionPattern.FindAllStringSubmatchIndex(content, -1)
	results := make([]apiFunction, 0, len(matches))
	for _, match := range matches {
		openBrace := match[0] + strings.LastIndex(content[match[0]:match[1]], "{")
		end := findMatchingBrace(content, openBrace)
		if end <= openBrace {
			continue
		}
		block := strings.TrimSpace(content[match[0] : end+1])
		controller := firstSubmatch(controllerNamePattern, block)
		methodsName := firstSubmatch(methodsNamePattern, block)
		if controller == "" && methodsName == "" {
			continue
		}

		results = append(results, apiFunction{
			FunctionName:     content[match[2]:match[3]],
			ControllerName:   controller,
			MethodsName:      methodsName,
			HTTPMethod:       strings.ToUpper(firstSubmatch(httpMethodPattern, block)),
			URL:              firstSubmatch(urlLiteralPattern, block),
			FilePath:         relPath,
			OriginalFilePath: relPath,
			ParamFields:      extractParamFields(block),
			Block:            ensureStatement(block),
			StartOffset:      match[0],
			EndOffset:        end + 1,
			StartLine:        lineNumberAtOffset(content, match[0]),
			EndLine:          lineNumberAtOffset(content, end+1),
		})
	}
	return results
}

func shouldSplitAPIModule(module apiModule) bool {
	if strings.HasPrefix(module.RelPath, "api/") || strings.Contains(module.RelPath, "/api/") {
		return false
	}

	controllers := make(map[string]struct{})
	for _, fn := range module.Functions {
		if fn.ControllerName != "" {
			controllers[fn.ControllerName] = struct{}{}
		}
	}
	return len(controllers) > 1
}

func splitAPIModule(rootDir string, module apiModule) (APISplitModule, []apiFunction, error) {
	groups := make(map[string][]apiFunction)
	for _, fn := range module.Functions {
		key := fn.ControllerName
		if key == "" {
			key = "unknown"
		}
		groups[key] = append(groups[key], fn)
	}

	usedTargets := make(map[string]struct{})
	for _, file := range mustCollectExistingFiles(rootDir) {
		usedTargets[file] = struct{}{}
	}

	controllers := make([]string, 0, len(groups))
	for controller := range groups {
		controllers = append(controllers, controller)
	}
	sort.Strings(controllers)

	split := APISplitModule{
		OriginalPath: module.RelPath,
		BarrelPath:   module.RelPath,
	}
	rewritten := make([]apiFunction, 0, len(module.Functions))
	exportsByTarget := make(map[string][]apiFunction)

	baseDir := path.Dir(module.RelPath)
	if baseDir == "." {
		baseDir = ""
	}
	for _, controller := range controllers {
		target := path.Join(baseDir, "api", controllerSlug(controller)+".js")
		target = allocateTargetName(target, usedTargets)
		usedTargets[target] = struct{}{}
		split.SplitPaths = append(split.SplitPaths, target)

		functions := groups[controller]
		for i := range functions {
			functions[i].OriginalFilePath = module.RelPath
			functions[i].FilePath = target
		}
		exportsByTarget[target] = functions
		rewritten = append(rewritten, functions...)
	}

	for _, target := range split.SplitPaths {
		content := buildSplitModuleContent(module, target, exportsByTarget[target])
		targetFull := filepath.Join(rootDir, filepath.FromSlash(target))
		if err := os.MkdirAll(filepath.Dir(targetFull), 0755); err != nil {
			return APISplitModule{}, nil, err
		}
		if err := os.WriteFile(targetFull, []byte(content), 0644); err != nil {
			return APISplitModule{}, nil, err
		}
	}

	if err := os.WriteFile(module.FullPath, []byte(buildBarrelModule(module.RelPath, exportsByTarget)), 0644); err != nil {
		return APISplitModule{}, nil, err
	}
	sort.Strings(split.SplitPaths)
	return split, rewritten, nil
}

func buildSplitModuleContent(module apiModule, target string, functions []apiFunction) string {
	var builder strings.Builder
	builder.WriteString("// 由 Gwxapkg 语义反混淆生成，来源: ")
	builder.WriteString(module.RelPath)
	builder.WriteString("\n")
	if module.Prefix != "" {
		builder.WriteString(rewritePrefixRequireForSplit(module.RelPath, target, module.Prefix))
		builder.WriteString("\n\n")
	}
	for _, fn := range functions {
		builder.WriteString(fn.Block)
		builder.WriteString("\n\n")
	}
	return strings.TrimSpace(builder.String()) + "\n"
}

func rewritePrefixRequireForSplit(sourceRel, targetRel, prefix string) string {
	return requireLiteralPattern.ReplaceAllStringFunc(prefix, func(match string) string {
		parts := requireLiteralPattern.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		quote := parts[1]
		literal := parts[2]
		resolved := resolveRequirePath(sourceRel, literal, nil)
		nextLiteral := buildRequireLiteral(targetRel, literal, resolved)
		return "require(" + quote + nextLiteral + quote + ")"
	})
}

func buildBarrelModule(sourceRel string, exportsByTarget map[string][]apiFunction) string {
	targets := make([]string, 0, len(exportsByTarget))
	for target := range exportsByTarget {
		targets = append(targets, target)
	}
	sort.Strings(targets)

	var builder strings.Builder
	builder.WriteString("// 由 Gwxapkg 语义反混淆生成的 API 兼容入口。\n")
	for index, target := range targets {
		alias := fmt.Sprintf("apiModule%d", index+1)
		builder.WriteString("var ")
		builder.WriteString(alias)
		builder.WriteString(" = require(\"")
		builder.WriteString(buildRequireLiteral(sourceRel, "./", target))
		builder.WriteString("\");\n")
		for _, fn := range exportsByTarget[target] {
			builder.WriteString("exports.")
			builder.WriteString(fn.FunctionName)
			builder.WriteString(" = ")
			builder.WriteString(alias)
			builder.WriteString(".")
			builder.WriteString(fn.FunctionName)
			builder.WriteString(";\n")
		}
	}
	return builder.String()
}

func collectReexports(rootDir string, jsFiles []string) (map[string]map[string]reexportTarget, error) {
	results := make(map[string]map[string]reexportTarget)
	for _, rel := range jsFiles {
		fullPath := filepath.Join(rootDir, filepath.FromSlash(rel))
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, err
		}
		text := string(data)
		aliases := parseRequireAliases(rel, text)
		for _, match := range exportReexportPattern.FindAllStringSubmatch(text, -1) {
			if len(match) != 4 {
				continue
			}
			targetModule, ok := aliases[match[2]]
			if !ok {
				continue
			}
			if results[rel] == nil {
				results[rel] = make(map[string]reexportTarget)
			}
			results[rel][match[1]] = reexportTarget{
				ModulePath:   targetModule,
				FunctionName: match[3],
			}
		}
	}
	return results, nil
}

func collectAPICallSites(rootDir string, jsFiles []string, endpoints []apiFunction, reexports map[string]map[string]reexportTarget) (map[string][]APICallSite, error) {
	endpointSet := make(map[string]struct{})
	for _, endpoint := range endpoints {
		endpointSet[endpointKey(endpoint.FilePath, endpoint.FunctionName)] = struct{}{}
	}

	results := make(map[string][]APICallSite)
	for _, rel := range jsFiles {
		if isAPIImplementationFile(rel, endpoints) {
			continue
		}
		fullPath := filepath.Join(rootDir, filepath.FromSlash(rel))
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, err
		}
		text := string(data)
		aliases := parseRequireAliases(rel, text)
		for alias, modulePath := range aliases {
			functions := candidateFunctionsForModule(modulePath, endpointSet, reexports)
			for exportName, target := range functions {
				callPattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(alias) + `\s*\.\s*` + regexp.QuoteMeta(exportName) + `\s*\(`)
				matches := callPattern.FindAllStringIndex(text, -1)
				for _, match := range matches {
					key := endpointKey(target.ModulePath, target.FunctionName)
					results[key] = append(results[key], APICallSite{
						FilePath:   rel,
						LineNumber: lineNumberAtOffset(text, match[0]),
						Expression: strings.TrimSpace(text[match[0]:match[1]]),
					})
				}
			}
		}
	}

	for key := range results {
		sort.Slice(results[key], func(i, j int) bool {
			if results[key][i].FilePath != results[key][j].FilePath {
				return results[key][i].FilePath < results[key][j].FilePath
			}
			return results[key][i].LineNumber < results[key][j].LineNumber
		})
	}
	return results, nil
}

func candidateFunctionsForModule(modulePath string, endpointSet map[string]struct{}, reexports map[string]map[string]reexportTarget) map[string]reexportTarget {
	results := make(map[string]reexportTarget)
	prefix := modulePath + "|"
	for key := range endpointSet {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		results[strings.TrimPrefix(key, prefix)] = reexportTarget{
			ModulePath:   modulePath,
			FunctionName: strings.TrimPrefix(key, prefix),
		}
	}
	for exportName, target := range reexports[modulePath] {
		if _, ok := endpointSet[endpointKey(target.ModulePath, target.FunctionName)]; ok {
			results[exportName] = target
		}
	}
	return results
}

func parseRequireAliases(currentRel, text string) map[string]string {
	results := make(map[string]string)
	for _, match := range requireAliasPattern.FindAllStringSubmatch(text, -1) {
		if len(match) != 3 {
			continue
		}
		results[match[1]] = resolveRequirePath(currentRel, match[2], nil)
	}
	return results
}

func isAPIImplementationFile(rel string, endpoints []apiFunction) bool {
	for _, endpoint := range endpoints {
		if rel == endpoint.FilePath || rel == endpoint.OriginalFilePath {
			return true
		}
	}
	return false
}

func writeAPIMap(rootDir string, report *APIMapReport) error {
	reportDir := filepath.Join(rootDir, reportDirName)
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(reportDir, "api_map.json"), data, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(reportDir, "api_map.md"), []byte(buildAPIMapMarkdown(report)), 0644); err != nil {
		return err
	}
	if err := writeAPICallChain(rootDir, report.CallChains); err != nil {
		return err
	}
	return writeAPIPseudo(rootDir, report)
}

func buildAPIMapMarkdown(report *APIMapReport) string {
	var builder strings.Builder
	builder.WriteString("# API 地图\n\n")
	builder.WriteString(fmt.Sprintf("- 生成时间: `%s`\n", report.GeneratedAt))
	builder.WriteString(fmt.Sprintf("- 接口函数: `%d`\n", len(report.Endpoints)))
	builder.WriteString(fmt.Sprintf("- 细拆模块: `%d`\n", report.SplitModuleCount))

	if len(report.SplitModules) > 0 {
		builder.WriteString("\n## 细拆模块\n\n")
		for _, split := range report.SplitModules {
			builder.WriteString(fmt.Sprintf("- `%s` -> %s\n", split.OriginalPath, inlineCodeList(split.SplitPaths)))
		}
	}

	builder.WriteString("\n## 接口清单\n\n")
	builder.WriteString("| 函数 | Controller | Method | HTTP | 文件 | 参数 | 调用点 |\n")
	builder.WriteString("|------|------------|--------|------|------|------|--------|\n")
	for _, endpoint := range report.Endpoints {
		calls := make([]string, 0, len(endpoint.CallSites))
		for _, call := range endpoint.CallSites {
			calls = append(calls, fmt.Sprintf("%s:%d", call.FilePath, call.LineNumber))
		}
		builder.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` | `%s` | %s | %s |\n",
			endpoint.FunctionName,
			endpoint.ControllerName,
			endpoint.MethodsName,
			emptyAsDash(endpoint.HTTPMethod),
			endpoint.FilePath,
			inlineCodeList(endpoint.ParamFields),
			inlineCodeList(calls),
		))
	}
	return builder.String()
}

func sortAPIMap(report *APIMapReport) {
	sort.Slice(report.SplitModules, func(i, j int) bool {
		return report.SplitModules[i].OriginalPath < report.SplitModules[j].OriginalPath
	})
	sort.Slice(report.Endpoints, func(i, j int) bool {
		if report.Endpoints[i].ControllerName != report.Endpoints[j].ControllerName {
			return report.Endpoints[i].ControllerName < report.Endpoints[j].ControllerName
		}
		if report.Endpoints[i].FilePath != report.Endpoints[j].FilePath {
			return report.Endpoints[i].FilePath < report.Endpoints[j].FilePath
		}
		return report.Endpoints[i].FunctionName < report.Endpoints[j].FunctionName
	})
}

func extractParamFields(block string) []string {
	match := firstObjectVarPattern.FindStringIndex(block)
	if len(match) != 2 {
		return nil
	}
	openBrace := strings.LastIndex(block[:match[1]], "{")
	if openBrace < 0 {
		return nil
	}
	end := findMatchingBrace(block, openBrace)
	if end <= openBrace {
		return nil
	}
	objectText := block[openBrace : end+1]
	fields := make([]string, 0)
	for _, match := range objectKeyPattern.FindAllStringSubmatch(objectText, -1) {
		if len(match) != 2 {
			continue
		}
		key := match[1]
		if key == "controllerName" || key == "methodsName" || key == "methodName" {
			continue
		}
		fields = append(fields, key)
	}
	return dedupeAndSort(fields)
}

func firstSubmatch(pattern *regexp.Regexp, text string) string {
	match := pattern.FindStringSubmatch(text)
	if len(match) != 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func ensureStatement(block string) string {
	block = strings.TrimSpace(block)
	if strings.HasSuffix(block, ";") {
		return block
	}
	return block + ";"
}

func endpointKey(filePath, functionName string) string {
	return filePath + "|" + functionName
}

func lineNumberAtOffset(text string, offset int) int {
	if offset < 0 {
		return 1
	}
	if offset > len(text) {
		offset = len(text)
	}
	line := 1
	for i := 0; i < offset; i++ {
		if text[i] == '\n' {
			line++
		}
	}
	return line
}

func inlineCodeList(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	escaped := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		escaped = append(escaped, "`"+strings.ReplaceAll(value, "`", "\\`")+"`")
	}
	if len(escaped) == 0 {
		return "-"
	}
	return strings.Join(escaped, "<br/>")
}

func emptyAsDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func mustCollectExistingFiles(rootDir string) []string {
	files, err := collectAllJSFiles(rootDir)
	if err != nil {
		return nil
	}
	return files
}
