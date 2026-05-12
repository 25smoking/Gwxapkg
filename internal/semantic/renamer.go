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
	"unicode"
)

const reportDirName = ".gwxapkg"

var (
	hashJSFilePattern       = regexp.MustCompile(`(?i)^[a-f0-9]{16,64}\.js$`)
	requireLiteralPattern   = regexp.MustCompile(`\brequire\s*\(\s*(["'])([^"']+\.js)["']\s*\)`)
	exportNamePattern       = regexp.MustCompile(`\bexports\.([A-Za-z_$][\w$]*)\s*=`)
	controllerNamePattern   = regexp.MustCompile(`\bcontrollerName\s*:\s*["']([^"']+)["']`)
	methodsNamePattern      = regexp.MustCompile(`\bmethods?Name\s*:\s*["']([^"']+)["']`)
	requestWrapperPattern   = regexp.MustCompile(`\bexports\.request\s*=\s*function\s*\(\s*([A-Za-z_$][\w$]*)\s*\)`)
	exportFunctionPattern   = regexp.MustCompile(`\bexports\.([A-Za-z_$][\w$]*)\s*=\s*function\s*\(\s*([A-Za-z_$][\w$]*)\s*\)\s*\{`)
	firstObjectVarPattern   = regexp.MustCompile(`\bvar\s+([A-Za-z_$][\w$]*)\s*=\s*\{`)
	sourceMapCommentPattern = regexp.MustCompile(`(?m)//[#@]\s*sourceMappingURL=(\S+)`)
)

// Report 描述语义化源码视图的生成结果。
type Report struct {
	GeneratedAt           string            `json:"generated_at"`
	RenamedCount          int               `json:"renamed_count"`
	RewrittenRequireCount int               `json:"rewritten_require_count"`
	SourceMapRecovered    int               `json:"source_map_recovered"`
	APISplitCount         int               `json:"api_split_count"`
	APIEndpointCount      int               `json:"api_endpoint_count"`
	APIMapJSONPath        string            `json:"api_map_json_path,omitempty"`
	APIMapMarkdownPath    string            `json:"api_map_markdown_path,omitempty"`
	APICallChainCount     int               `json:"api_call_chain_count"`
	APICallChainJSONPath  string            `json:"api_call_chain_json_path,omitempty"`
	APICallChainMDPath    string            `json:"api_call_chain_md_path,omitempty"`
	APIPseudoMarkdownPath string            `json:"api_pseudo_markdown_path,omitempty"`
	ASTRenamedCount       int               `json:"ast_renamed_count"`
	ASTRenamedFiles       int               `json:"ast_renamed_files"`
	ASTRenameMapPath      string            `json:"ast_rename_map_path,omitempty"`
	Modules               []ModuleReport    `json:"modules"`
	SourceMaps            []SourceMapReport `json:"source_maps,omitempty"`
	PathMap               map[string]string `json:"path_map"`
}

// ModuleReport 记录一个模块从编译名到语义名的映射。
type ModuleReport struct {
	OriginalPath string   `json:"original_path"`
	SemanticPath string   `json:"semantic_path"`
	Role         string   `json:"role"`
	Reason       string   `json:"reason"`
	Exports      []string `json:"exports,omitempty"`
	Controllers  []string `json:"controllers,omitempty"`
	Methods      []string `json:"methods,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
}

// SourceMapReport 记录通过 source map 恢复出的原始源码。
type SourceMapReport struct {
	MapPath        string   `json:"map_path"`
	OutputRoot     string   `json:"output_root"`
	RecoveredFiles []string `json:"recovered_files"`
}

type moduleInfo struct {
	relPath      string
	fullPath     string
	content      string
	exports      []string
	controllers  []string
	methods      []string
	dependencies []string
	role         string
	reason       string
	targetRel    string
}

type sourceMapFile struct {
	Sources        []string `json:"sources"`
	SourcesContent []string `json:"sourcesContent"`
}

// RewriteOptions 控制 semantic 后处理链路。
type RewriteOptions struct {
	ASTRename ASTRenameOptions
}

// DefaultRewriteOptions 返回默认 deep AST 还原配置。
func DefaultRewriteOptions() RewriteOptions {
	return RewriteOptions{ASTRename: DefaultASTRenameOptions()}
}

// RewriteProject 将编译后的哈希模块改写为更适合审计的源码视图。
//
// 这个过程只处理已还原工程目录里的 JS 模块：
//  1. 识别 16-64 位十六进制文件名的模块；
//  2. 根据 exports、controllerName、methodsName、请求封装等线索推断语义文件名；
//  3. 重写所有 require 路径，保证重命名后仍可追踪依赖；
//  4. 对常见接口封装做小范围变量语义化；
//  5. 如果存在 source map，则把 sourcesContent 落到 .gwxapkg/sources 下。
func RewriteProject(rootDir string) (*Report, error) {
	return RewriteProjectWithOptions(rootDir, DefaultRewriteOptions())
}

// RewriteProjectWithOptions 将编译后的哈希模块按配置改写为更适合审计的源码视图。
func RewriteProjectWithOptions(rootDir string, options RewriteOptions) (*Report, error) {
	rootAbs, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("解析输出目录失败: %w", err)
	}
	if strings.TrimSpace(options.ASTRename.Mode) == "" {
		options.ASTRename = DefaultASTRenameOptions()
	} else {
		options.ASTRename = normalizeASTRenameOptions(options.ASTRename)
	}

	modules, allJSFiles, err := collectModules(rootAbs)
	if err != nil {
		return nil, err
	}
	if len(modules) == 0 {
		report := &Report{
			GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
			PathMap:     map[string]string{},
		}
		if existing, ok := readExistingReport(rootAbs); ok {
			report = existing
			report.GeneratedAt = time.Now().Format("2006-01-02 15:04:05")
			if report.PathMap == nil {
				report.PathMap = map[string]string{}
			}
		}

		apiMap, err := BuildAPIMap(rootAbs, allJSFiles)
		if err != nil {
			return nil, err
		}
		attachAPIMapReport(report, apiMap)

		latestJSFiles, err := collectAllJSFiles(rootAbs)
		if err != nil {
			return nil, err
		}
		if options.ASTRename.Mode != ASTRenameModeOff {
			astReport, err := RenameIdentifiersWithOptions(rootAbs, latestJSFiles, options.ASTRename)
			if err != nil {
				return nil, err
			}
			attachASTRenameReport(report, astReport)
		}

		sourceReports, err := recoverSourceMaps(rootAbs, latestJSFiles)
		if err != nil {
			return nil, err
		}
		report.SourceMaps = sourceReports
		for _, sourceReport := range sourceReports {
			report.SourceMapRecovered += len(sourceReport.RecoveredFiles)
		}
		_ = writeReport(rootAbs, report)
		return report, nil
	}

	pathMap := buildRenamePlan(modules, allJSFiles)
	report := &Report{
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		PathMap:     pathMap,
	}

	if len(pathMap) > 0 {
		rewritten, err := rewriteRequiresAndSource(rootAbs, allJSFiles, pathMap, modules)
		if err != nil {
			return nil, err
		}
		report.RewrittenRequireCount = rewritten

		if err := renameFiles(rootAbs, pathMap); err != nil {
			return nil, err
		}
	}

	for _, module := range modules {
		semanticPath := module.relPath
		if next, ok := pathMap[module.relPath]; ok {
			semanticPath = next
			report.RenamedCount++
		}
		report.Modules = append(report.Modules, ModuleReport{
			OriginalPath: module.relPath,
			SemanticPath: semanticPath,
			Role:         module.role,
			Reason:       module.reason,
			Exports:      module.exports,
			Controllers:  module.controllers,
			Methods:      module.methods,
			Dependencies: module.dependencies,
		})
	}
	sort.Slice(report.Modules, func(i, j int) bool {
		return report.Modules[i].OriginalPath < report.Modules[j].OriginalPath
	})

	finalJSFiles := remapJSFiles(allJSFiles, pathMap)
	apiMap, err := BuildAPIMap(rootAbs, finalJSFiles)
	if err != nil {
		return nil, err
	}
	attachAPIMapReport(report, apiMap)

	latestJSFiles, err := collectAllJSFiles(rootAbs)
	if err != nil {
		return nil, err
	}
	if options.ASTRename.Mode != ASTRenameModeOff {
		astReport, err := RenameIdentifiersWithOptions(rootAbs, latestJSFiles, options.ASTRename)
		if err != nil {
			return nil, err
		}
		attachASTRenameReport(report, astReport)
	}

	sourceReports, err := recoverSourceMaps(rootAbs, latestJSFiles)
	if err != nil {
		return nil, err
	}
	report.SourceMaps = sourceReports
	for _, sourceReport := range sourceReports {
		report.SourceMapRecovered += len(sourceReport.RecoveredFiles)
	}

	if err := writeReport(rootAbs, report); err != nil {
		return nil, err
	}
	return report, nil
}

func attachAPIMapReport(report *Report, apiMap *APIMapReport) {
	if report == nil || apiMap == nil {
		return
	}
	report.APISplitCount = apiMap.SplitModuleCount
	report.APIEndpointCount = len(apiMap.Endpoints)
	report.APIMapJSONPath = path.Join(reportDirName, "api_map.json")
	report.APIMapMarkdownPath = path.Join(reportDirName, "api_map.md")
	report.APICallChainCount = len(apiMap.CallChains)
	report.APICallChainJSONPath = path.Join(reportDirName, apiCallChainJSONFileName)
	report.APICallChainMDPath = path.Join(reportDirName, apiCallChainMDFileName)
	report.APIPseudoMarkdownPath = path.Join(reportDirName, apiPseudoMDFileName)
}

func attachASTRenameReport(report *Report, astReport *ASTRenameReport) {
	if report == nil || astReport == nil {
		return
	}
	report.ASTRenamedCount = astReport.TotalRenames
	report.ASTRenamedFiles = astReport.RenamedFiles
	report.ASTRenameMapPath = path.Join(reportDirName, astRenameReportFileName)
}

func remapJSFiles(files []string, pathMap map[string]string) []string {
	results := make([]string, 0, len(files))
	for _, file := range files {
		if next, ok := pathMap[file]; ok {
			results = append(results, next)
			continue
		}
		results = append(results, file)
	}
	sort.Strings(results)
	return results
}

func collectModules(rootDir string) ([]*moduleInfo, []string, error) {
	modules := make([]*moduleInfo, 0)
	allJSFiles := make([]string, 0)

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
		rel = filepath.ToSlash(rel)
		allJSFiles = append(allJSFiles, rel)

		if !isHashModule(rel) {
			return nil
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		modules = append(modules, analyzeModule(rel, filePath, string(data)))
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("扫描 JS 模块失败: %w", err)
	}

	sort.Strings(allJSFiles)
	sort.Slice(modules, func(i, j int) bool {
		return modules[i].relPath < modules[j].relPath
	})
	return modules, allJSFiles, nil
}

func shouldSkipDir(rootDir, filePath string) bool {
	rel, err := filepath.Rel(rootDir, filePath)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	return rel == reportDirName || strings.HasPrefix(rel, reportDirName+"/")
}

func isHashModule(relPath string) bool {
	return hashJSFilePattern.MatchString(path.Base(relPath))
}

func analyzeModule(relPath, fullPath, content string) *moduleInfo {
	info := &moduleInfo{
		relPath:      relPath,
		fullPath:     fullPath,
		content:      content,
		exports:      uniqueMatches(exportNamePattern, content),
		controllers:  uniqueMatches(controllerNamePattern, content),
		methods:      uniqueMatches(methodsNamePattern, content),
		dependencies: uniqueRequires(content),
	}
	info.role, info.reason = inferRole(info, content)
	return info
}

func buildRenamePlan(modules []*moduleInfo, allJSFiles []string) map[string]string {
	used := make(map[string]struct{}, len(modules))
	pathMap := make(map[string]string, len(modules))
	modulePaths := make(map[string]struct{}, len(modules))
	for _, module := range modules {
		modulePaths[module.relPath] = struct{}{}
	}
	for _, rel := range allJSFiles {
		if _, isModule := modulePaths[rel]; !isModule {
			used[rel] = struct{}{}
		}
	}

	for _, module := range modules {
		dir := path.Dir(module.relPath)
		if dir == "." {
			dir = ""
		}
		baseName := inferBaseName(module)
		candidate := baseName + ".js"
		if dir != "" {
			candidate = path.Join(dir, candidate)
		}
		candidate = allocateTargetName(candidate, used)
		used[candidate] = struct{}{}
		module.targetRel = candidate
		if candidate != module.relPath {
			pathMap[module.relPath] = candidate
		}
	}

	return pathMap
}

func allocateTargetName(candidate string, used map[string]struct{}) string {
	if _, exists := used[candidate]; !exists {
		return candidate
	}
	dir, file := path.Split(candidate)
	ext := path.Ext(file)
	base := strings.TrimSuffix(file, ext)
	for i := 2; ; i++ {
		next := path.Join(dir, fmt.Sprintf("%s_%d%s", base, i, ext))
		if _, exists := used[next]; !exists {
			return next
		}
	}
}

func rewriteRequiresAndSource(rootDir string, allJSFiles []string, pathMap map[string]string, modules []*moduleInfo) (int, error) {
	moduleByPath := make(map[string]*moduleInfo, len(modules))
	for _, module := range modules {
		moduleByPath[module.relPath] = module
	}

	rewrittenCount := 0
	for _, rel := range allJSFiles {
		fullPath := filepath.Join(rootDir, filepath.FromSlash(rel))
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return 0, err
		}
		content := string(data)
		changed := false

		content = requireLiteralPattern.ReplaceAllStringFunc(content, func(match string) string {
			parts := requireLiteralPattern.FindStringSubmatch(match)
			if len(parts) != 3 {
				return match
			}
			quote := parts[1]
			literal := parts[2]
			resolved := resolveRequirePath(rel, literal, pathMap)
			target, ok := pathMap[resolved]
			if !ok {
				return match
			}
			nextLiteral := buildRequireLiteral(rel, literal, target)
			rewrittenCount++
			changed = true
			return "require(" + quote + nextLiteral + quote + ")"
		})

		if module, ok := moduleByPath[rel]; ok {
			next, semanticChanged := rewriteSemanticVariables(content, module)
			if semanticChanged {
				content = next
				changed = true
			}
		}

		if changed {
			if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
				return 0, err
			}
		}
	}

	return rewrittenCount, nil
}

func resolveRequirePath(currentRel, literal string, pathMap map[string]string) string {
	currentDir := path.Dir(currentRel)
	if currentDir == "." {
		currentDir = ""
	}

	var candidate string
	if strings.HasPrefix(literal, ".") {
		candidate = path.Clean(path.Join(currentDir, literal))
	} else {
		candidate = path.Clean(path.Join(currentDir, literal))
		if _, ok := pathMap[candidate]; !ok {
			candidate = path.Clean(literal)
		}
	}
	return strings.TrimPrefix(candidate, "./")
}

func buildRequireLiteral(currentRel, originalLiteral, targetRel string) string {
	currentDir := path.Dir(currentRel)
	if currentDir == "." {
		currentDir = ""
	}

	if !strings.HasPrefix(originalLiteral, ".") && currentDir == "" {
		return targetRel
	}

	rel, err := filepath.Rel(filepath.FromSlash(currentDir), filepath.FromSlash(targetRel))
	if err != nil {
		return targetRel
	}
	next := filepath.ToSlash(rel)
	if !strings.HasPrefix(next, ".") {
		next = "./" + next
	}
	return next
}

func renameFiles(rootDir string, pathMap map[string]string) error {
	type move struct {
		from string
		to   string
		tmp  string
	}

	moves := make([]move, 0, len(pathMap))
	for from, to := range pathMap {
		fromFull := filepath.Join(rootDir, filepath.FromSlash(from))
		toFull := filepath.Join(rootDir, filepath.FromSlash(to))
		if err := os.MkdirAll(filepath.Dir(toFull), 0755); err != nil {
			return err
		}
		tmpFull := fromFull + ".gwxrename"
		moves = append(moves, move{from: fromFull, to: toFull, tmp: tmpFull})
	}

	for _, item := range moves {
		if err := os.Rename(item.from, item.tmp); err != nil {
			return fmt.Errorf("准备重命名 %s 失败: %w", item.from, err)
		}
	}
	for _, item := range moves {
		if err := os.Rename(item.tmp, item.to); err != nil {
			return fmt.Errorf("重命名到 %s 失败: %w", item.to, err)
		}
	}
	return nil
}

func rewriteSemanticVariables(content string, module *moduleInfo) (string, bool) {
	changed := false
	if module.role == "api" {
		next, ok := rewriteAPIWrapperVariables(content)
		if ok {
			content = next
			changed = true
		}
	}
	if module.role == "request" {
		next, ok := rewriteRequestWrapperVariables(content)
		if ok {
			content = next
			changed = true
		}
	}
	return content, changed
}

func rewriteAPIWrapperVariables(content string) (string, bool) {
	matches := exportFunctionPattern.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return content, false
	}

	var builder strings.Builder
	builder.Grow(len(content))
	last := 0
	changed := false

	for _, match := range matches {
		openBrace := match[0] + strings.LastIndex(content[match[0]:match[1]], "{")
		end := findMatchingBrace(content, openBrace)
		if end <= openBrace {
			continue
		}

		block := content[match[0] : end+1]
		if !strings.Contains(block, "controllerName") || !strings.Contains(block, "methodsName") {
			continue
		}

		params := content[match[4]:match[5]]
		if !isShortIdentifier(params) {
			continue
		}
		varName := ""
		if varMatch := firstObjectVarPattern.FindStringSubmatch(block); len(varMatch) == 2 && isShortIdentifier(varMatch[1]) {
			varName = varMatch[1]
		}

		updated := regexp.MustCompile(`function\s*\(\s*`+regexp.QuoteMeta(params)+`\s*\)`).ReplaceAllString(block, "function(params)")
		updated = regexp.MustCompile(`\b`+regexp.QuoteMeta(params)+`\.`).ReplaceAllString(updated, "params.")
		if varName != "" {
			updated = regexp.MustCompile(`\bvar\s+`+regexp.QuoteMeta(varName)+`\s*=`).ReplaceAllString(updated, "var requestData =")
			updated = regexp.MustCompile(`\bdata\s*:\s*`+regexp.QuoteMeta(varName)+`\b`).ReplaceAllString(updated, "data: requestData")
		}

		if updated == block {
			continue
		}

		builder.WriteString(content[last:match[0]])
		builder.WriteString(updated)
		last = end + 1
		changed = true
	}

	if !changed {
		return content, false
	}
	builder.WriteString(content[last:])
	return builder.String(), true
}

func rewriteRequestWrapperVariables(content string) (string, bool) {
	match := requestWrapperPattern.FindStringSubmatchIndex(content)
	if len(match) < 4 {
		return content, false
	}
	param := content[match[2]:match[3]]
	if !isShortIdentifier(param) {
		return content, false
	}
	openBrace := strings.Index(content[match[0]:], "{")
	if openBrace < 0 {
		return content, false
	}
	openBrace += match[0]
	end := findMatchingBrace(content, openBrace)
	if end <= openBrace {
		return content, false
	}
	block := content[match[0] : end+1]
	updated := regexp.MustCompile(`function\s*\(\s*`+regexp.QuoteMeta(param)+`\s*\)`).ReplaceAllString(block, "function(options)")
	updated = regexp.MustCompile(`\b`+regexp.QuoteMeta(param)+`\.`).ReplaceAllString(updated, "options.")
	if updated == block {
		return content, false
	}
	return content[:match[0]] + updated + content[end+1:], true
}

func findMatchingBrace(content string, openBrace int) int {
	depth := 0
	inSingle, inDouble, inTemplate := false, false, false
	inLineComment, inBlockComment := false, false

	for i := openBrace; i < len(content); i++ {
		ch := content[i]
		if inLineComment {
			if ch == '\n' {
				inLineComment = false
			}
			continue
		}
		if inBlockComment {
			if ch == '*' && i+1 < len(content) && content[i+1] == '/' {
				i++
				inBlockComment = false
			}
			continue
		}
		if inSingle || inDouble || inTemplate {
			if ch == '\\' {
				i++
				continue
			}
			if inSingle && ch == '\'' {
				inSingle = false
			} else if inDouble && ch == '"' {
				inDouble = false
			} else if inTemplate && ch == '`' {
				inTemplate = false
			}
			continue
		}
		if ch == '/' && i+1 < len(content) {
			if content[i+1] == '/' {
				i++
				inLineComment = true
				continue
			}
			if content[i+1] == '*' {
				i++
				inBlockComment = true
				continue
			}
		}
		switch ch {
		case '\'':
			inSingle = true
		case '"':
			inDouble = true
		case '`':
			inTemplate = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func inferRole(module *moduleInfo, content string) (string, string) {
	hasExport := func(name string) bool {
		for _, exportName := range module.exports {
			if exportName == name {
				return true
			}
		}
		return false
	}

	switch {
	case hasExport("request"):
		return "request", "封装 wx/uni request 调用"
	case hasExport("config") || strings.Contains(content, "baseApiUrl") || strings.Contains(content, "baseImgUrl"):
		return "config", "包含 baseApiUrl/baseImgUrl 或 exports.config"
	case len(module.controllers) > 0 || len(module.methods) > 0:
		return "api", "包含 controllerName/methodsName 接口调度参数"
	case strings.Contains(content, ".request({"):
		return "request", "封装 wx/uni request 调用"
	case hasExport("Encrypt") || strings.Contains(content, "CryptoJS") || strings.Contains(content, "sm2"):
		return "crypto", "包含加密/编码相关导出"
	case strings.Contains(content, "createSSRApp") || strings.Contains(content, "exports.index") && strings.Contains(content, "wx$1"):
		return "runtime", "包含 uni/vue 运行时导出"
	case hasExport("VantComponent"):
		return "ui", "Vant 组件工厂"
	case containsAny(module.exports, "MiniProgramPage"):
		return "page-factory", "小程序页面工厂"
	case containsAny(module.exports, "fontData"):
		return "asset", "图标字体数据"
	case containsAny(module.exports, "share"):
		return "config", "分享配置"
	case containsAny(module.exports, "formatDate2", "formatDate_cn", "formatChatTime"):
		return "util", "日期时间格式化工具"
	case containsAny(module.exports, "getTabBar"):
		return "util", "TabBar 工具"
	case strings.Contains(content, "_imports_"):
		return "asset", "静态资源引用表"
	default:
		return "module", "根据 exports 推断的普通模块"
	}
}

func inferBaseName(module *moduleInfo) string {
	switch module.role {
	case "config":
		if containsAny(module.exports, "share") {
			return "share_config"
		}
		return "config"
	case "request":
		return "request"
	case "api":
		return inferAPIBaseName(module)
	case "crypto":
		return "crypto_encrypt"
	case "runtime":
		return "vendor_uni_runtime"
	case "ui":
		if containsAny(module.exports, "VantComponent") {
			return "ui_vant_component"
		}
		return "ui_component"
	case "page-factory":
		return "mini_program_page"
	case "asset":
		if containsAny(module.exports, "fontData") {
			return "uni_icon_font"
		}
		return "static_asset_refs"
	case "util":
		if containsAny(module.exports, "getTabBar") {
			return "utils_tabbar"
		}
		if containsAny(module.exports, "formatDate2", "formatDate_cn", "formatChatTime") {
			return "utils_date"
		}
		return "utils_" + firstExportSlug(module)
	default:
		if containsAny(module.exports, "clone") {
			return "utils_clone"
		}
		if containsAny(module.exports, "escape2Html", "html2Escape") {
			return "utils_html_escape"
		}
		if containsAny(module.exports, "handleDataset") {
			return "utils_dataset"
		}
		if containsAny(module.exports, "getRelationNodes") {
			return "utils_relation_nodes"
		}
		if containsAny(module.exports, "button", "link", "pickerProps", "transition") {
			return "ui_" + firstExportSlug(module)
		}
		return "module_" + firstExportSlug(module)
	}
}

func inferAPIBaseName(module *moduleInfo) string {
	controllerSlugs := make([]string, 0, len(module.controllers))
	for _, controller := range module.controllers {
		controllerSlugs = append(controllerSlugs, controllerSlug(controller))
	}
	controllerSlugs = dedupeAndSort(controllerSlugs)

	if containsAny(controllerSlugs, "login", "user") && containsMethodFragment(module.methods, "login", "password", "sms", "register") {
		return "api_auth"
	}
	if len(controllerSlugs) > 0 {
		if len(controllerSlugs) > 3 {
			controllerSlugs = controllerSlugs[:3]
		}
		return "api_" + strings.Join(controllerSlugs, "_")
	}

	methodSlugs := make([]string, 0, len(module.methods))
	for _, method := range module.methods {
		methodSlugs = append(methodSlugs, toSnakeCase(method))
	}
	methodSlugs = dedupeAndSort(methodSlugs)
	if len(methodSlugs) > 0 {
		return "api_" + methodSlugs[0]
	}
	return "api_" + firstExportSlug(module)
}

func controllerSlug(controller string) string {
	switch strings.ToLower(controller) {
	case "cerinfo":
		return "cert"
	case "drivinglicense", "drivinglicence":
		return "driving_license"
	case "login", "user":
		return strings.ToLower(controller)
	default:
		return toSnakeCase(controller)
	}
}

func firstExportSlug(module *moduleInfo) string {
	if len(module.exports) == 0 {
		return strings.TrimSuffix(strings.ToLower(path.Base(module.relPath)), ".js")
	}
	return toSnakeCase(module.exports[0])
}

func recoverSourceMaps(rootDir string, jsFiles []string) ([]SourceMapReport, error) {
	reports := make([]SourceMapReport, 0)
	seenMaps := map[string]struct{}{}

	for _, rel := range jsFiles {
		fullPath := filepath.Join(rootDir, filepath.FromSlash(rel))
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, err
		}
		for _, match := range sourceMapCommentPattern.FindAllStringSubmatch(string(data), -1) {
			if len(match) != 2 || strings.HasPrefix(match[1], "data:") {
				continue
			}
			mapRel := path.Clean(path.Join(path.Dir(rel), match[1]))
			if _, exists := seenMaps[mapRel]; exists {
				continue
			}
			seenMaps[mapRel] = struct{}{}

			report, err := recoverOneSourceMap(rootDir, mapRel)
			if err != nil {
				continue
			}
			if len(report.RecoveredFiles) > 0 {
				reports = append(reports, report)
			}
		}
	}
	return reports, nil
}

func recoverOneSourceMap(rootDir, mapRel string) (SourceMapReport, error) {
	mapFull := filepath.Join(rootDir, filepath.FromSlash(mapRel))
	data, err := os.ReadFile(mapFull)
	if err != nil {
		return SourceMapReport{}, err
	}

	var parsed sourceMapFile
	if err := json.Unmarshal(data, &parsed); err != nil {
		return SourceMapReport{}, err
	}
	if len(parsed.Sources) == 0 || len(parsed.SourcesContent) == 0 {
		return SourceMapReport{}, nil
	}

	outputRootRel := path.Join(reportDirName, "sources", strings.TrimSuffix(mapRel, path.Ext(mapRel)))
	report := SourceMapReport{
		MapPath:    mapRel,
		OutputRoot: outputRootRel,
	}

	for index, sourceName := range parsed.Sources {
		if index >= len(parsed.SourcesContent) || parsed.SourcesContent[index] == "" {
			continue
		}
		safeName := sanitizeSourcePath(sourceName)
		if safeName == "" {
			continue
		}
		targetRel := path.Join(outputRootRel, safeName)
		targetFull := filepath.Join(rootDir, filepath.FromSlash(targetRel))
		if err := os.MkdirAll(filepath.Dir(targetFull), 0755); err != nil {
			return SourceMapReport{}, err
		}
		if err := os.WriteFile(targetFull, []byte(parsed.SourcesContent[index]), 0644); err != nil {
			return SourceMapReport{}, err
		}
		report.RecoveredFiles = append(report.RecoveredFiles, targetRel)
	}
	return report, nil
}

func sanitizeSourcePath(sourceName string) string {
	sourceName = strings.TrimSpace(strings.ReplaceAll(sourceName, "\\", "/"))
	sourceName = strings.TrimPrefix(sourceName, "webpack://")
	sourceName = strings.TrimPrefix(sourceName, "file://")
	sourceName = strings.TrimLeft(sourceName, "/")
	sourceName = path.Clean(sourceName)
	if sourceName == "." || sourceName == "" || sourceName == ".." || strings.HasPrefix(sourceName, "../") {
		return ""
	}
	sourceName = strings.ReplaceAll(sourceName, ":", "_")
	return sourceName
}

func writeReport(rootDir string, report *Report) error {
	reportDir := filepath.Join(rootDir, reportDirName)
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(reportDir, "semantic_module_map.json"), data, 0644)
}

func readExistingReport(rootDir string) (*Report, bool) {
	data, err := os.ReadFile(filepath.Join(rootDir, reportDirName, "semantic_module_map.json"))
	if err != nil {
		return nil, false
	}
	var report Report
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, false
	}
	return &report, true
}

func uniqueMatches(pattern *regexp.Regexp, text string) []string {
	matches := pattern.FindAllStringSubmatch(text, -1)
	values := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) == 2 {
			values = append(values, match[1])
		}
	}
	return dedupeAndSort(values)
}

func uniqueRequires(text string) []string {
	matches := requireLiteralPattern.FindAllStringSubmatch(text, -1)
	values := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) == 3 {
			values = append(values, match[2])
		}
	}
	return dedupeAndSort(values)
}

func containsAny(values []string, needles ...string) bool {
	needleSet := make(map[string]struct{}, len(needles))
	for _, needle := range needles {
		needleSet[strings.ToLower(needle)] = struct{}{}
	}
	for _, value := range values {
		if _, ok := needleSet[strings.ToLower(value)]; ok {
			return true
		}
	}
	return false
}

func containsMethodFragment(methods []string, fragments ...string) bool {
	for _, method := range methods {
		method = strings.ToLower(method)
		for _, fragment := range fragments {
			if strings.Contains(method, strings.ToLower(fragment)) {
				return true
			}
		}
	}
	return false
}

func dedupeAndSort(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	results := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		results = append(results, value)
	}
	sort.Strings(results)
	return results
}

func toSnakeCase(value string) string {
	var builder strings.Builder
	var previousUnderscore bool
	var previousLowerOrDigit bool
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if unicode.IsUpper(r) && previousLowerOrDigit && !previousUnderscore {
				builder.WriteByte('_')
			}
			builder.WriteRune(unicode.ToLower(r))
			previousUnderscore = false
			previousLowerOrDigit = unicode.IsLower(r) || unicode.IsDigit(r)
			continue
		}
		if builder.Len() > 0 && !previousUnderscore {
			builder.WriteByte('_')
			previousUnderscore = true
		}
		previousLowerOrDigit = false
	}
	return strings.Trim(builder.String(), "_")
}

func isShortIdentifier(value string) bool {
	if value == "" || len(value) > 2 {
		return false
	}
	for _, r := range value {
		if !(unicode.IsLetter(r) || r == '_' || r == '$') {
			return false
		}
	}
	return true
}
