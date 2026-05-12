package analyzer

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/25smoking/Gwxapkg/internal/scanner"
)

type appConfig struct {
	Pages                          []string               `json:"pages"`
	EntryPagePath                  string                 `json:"entryPagePath"`
	Window                         map[string]interface{} `json:"window"`
	Global                         map[string]interface{} `json:"global"`
	TabBar                         map[string]interface{} `json:"tabBar"`
	SubPackages                    []subPackageConfig     `json:"subPackages"`
	Subpackages                    []subPackageConfig     `json:"subpackages"`
	NavigateToMiniProgramAppIdList []string               `json:"navigateToMiniProgramAppIdList"`
}

type subPackageConfig struct {
	Root  string   `json:"root"`
	Pages []string `json:"pages"`
}

type pageJSONConfig struct {
	NavigationBarTitleText string            `json:"navigationBarTitleText"`
	UsingComponents        map[string]string `json:"usingComponents"`
	Component              bool              `json:"component"`
}

type jsHandler struct {
	Name  string
	Start int
	End   int
	Body  string
}

type wxmlAction struct {
	Tag          string
	TriggerEvent string
	HandlerName  string
	RawTarget    string
	TriggerText  string
	LineNumber   int
	SourceFile   string
}

var (
	jsNavigationCallStartPattern = regexp.MustCompile(`(?is)\b(?:wx|uni|tt|my)\.(navigateTo|redirectTo|reLaunch|switchTab)\s*\(\s*\{`)
	jsURLLiteralPattern          = regexp.MustCompile("(?s)[\"'`](.*?)[\"'`]")
	requirePattern               = regexp.MustCompile("(?m)\\brequire\\(\\s*[\"'`]([^\"'`]+)[\"'`]\\s*\\)")
	importPattern                = regexp.MustCompile("(?m)\\bimport\\s+(?:[^;\\n]*?\\s+from\\s+)?[\"'`]([^\"'`]+)[\"'`]")
	methodPatternFunc            = regexp.MustCompile(`(?m)(?:^|[,{]\s*)([A-Za-z_$][\w$]*)\s*:\s*function\s*\(([^\n)]*)\)\s*\{`)
	methodPatternArrow           = regexp.MustCompile(`(?m)(?:^|[,{]\s*)([A-Za-z_$][\w$]*)\s*:\s*\(([^\n)]*)\)\s*=>\s*\{`)
	methodPatternShort           = regexp.MustCompile(`(?m)^\s*([A-Za-z_$][\w$]*)\s*\(([^\n)]*)\)\s*\{`)
	wxmlNavigatorBlockPattern    = regexp.MustCompile(`(?is)<navigator\b([^>]*)>(.*?)</navigator>`)
	wxmlActionBlockPattern       = regexp.MustCompile(`(?is)<([a-zA-Z0-9:_-]+)\b([^>]*)>(.*?)</[a-zA-Z0-9:_-]+>`)
	wxmlActionSelfClosePattern   = regexp.MustCompile(`(?is)<([a-zA-Z0-9:_-]+)\b([^>]*)/>`)
	attrURLPattern               = regexp.MustCompile("(?is)\\burl\\s*=\\s*[\"'`]([^\"'`]+)[\"'`]")
	attrOpenTypePattern          = regexp.MustCompile("(?is)\\bopen-type\\s*=\\s*[\"'`]([^\"'`]+)[\"'`]")
	attrEventPattern             = regexp.MustCompile("(?is)\\b(bindtap|catchtap|bind:tap|catch:tap|capture-bind:tap|capture-catch:tap)\\s*=\\s*[\"'`]([^\"'`]+)[\"'`]")
	attrDataTargetPattern        = regexp.MustCompile("(?is)\\bdata-(url|route|path|page)\\s*=\\s*[\"'`]([^\"'`]+)[\"'`]")
	pageScriptPattern            = regexp.MustCompile(`(?m)\bPage\s*\(`)
	versionAPIPathPattern        = regexp.MustCompile(`^v\d+/`)
)

// AnalyzeMiniProgram 分析已解包目录中的页面与路由关系。
func AnalyzeMiniProgram(rootDir, appID string) (*RouteManifest, error) {
	app, configSource, err := loadAppConfig(rootDir)
	if err != nil {
		return nil, err
	}

	entryPage := normalizeRoute(app.EntryPagePath)
	if entryPage == "" && len(app.Pages) > 0 {
		entryPage = normalizeRoute(app.Pages[0])
	}

	manifest := &RouteManifest{
		AppID:                appID,
		ConfigSource:         configSource,
		GeneratedAt:          time.Now().Format("2006-01-02 15:04:05"),
		EntryPage:            entryPage,
		ExternalMiniPrograms: dedupeAndSortStrings(app.NavigateToMiniProgramAppIdList),
		Pages:                make([]PageNode, 0),
		NavigationEdges:      make([]NavigationEdge, 0),
		OrphanPages:          make([]string, 0),
	}
	analyzerCtx := newRouteAnalyzerContext(rootDir)

	globalWindow := app.Window
	if len(globalWindow) == 0 && len(app.Global) > 0 {
		if window, ok := app.Global["window"].(map[string]interface{}); ok {
			globalWindow = window
		}
	}
	globalTitle := stringFromMap(globalWindow, "navigationBarTitleText")

	tabBarItems, tabBarSet := extractTabBar(app.TabBar)
	manifest.TabBar = tabBarItems

	pageIndex := make(map[string]*PageNode)
	pageRoutes := make([]string, 0)
	addPage := func(route, packageType, packageRoot string) {
		normalized := normalizeRoute(route)
		if normalized == "" {
			return
		}
		if _, exists := pageIndex[normalized]; exists {
			return
		}
		node := &PageNode{
			Route:       normalized,
			PackageType: packageType,
			PackageRoot: packageRoot,
			IsEntry:     normalized == entryPage,
			IsTabBar:    tabBarSet[normalized],
		}
		pageIndex[normalized] = node
		pageRoutes = append(pageRoutes, normalized)
	}

	for _, page := range app.Pages {
		addPage(page, "main", "")
	}

	subPackages := app.SubPackages
	if len(app.Subpackages) > 0 {
		subPackages = append(subPackages, app.Subpackages...)
	}
	for _, subPackage := range subPackages {
		root := normalizeRoute(subPackage.Root)
		pages := make([]string, 0, len(subPackage.Pages))
		for _, page := range subPackage.Pages {
			fullRoute := joinRoute(root, page)
			addPage(fullRoute, "subpackage", root)
			if normalized := normalizeRoute(fullRoute); normalized != "" {
				pages = append(pages, normalized)
			}
		}
		pages = dedupeAndSortStrings(pages)
		manifest.SubPackages = append(manifest.SubPackages, SubPackageInfo{
			Root:      root,
			PageCount: len(pages),
			Pages:     pages,
		})
	}

	sort.Strings(pageRoutes)
	for _, route := range pageRoutes {
		node := pageIndex[route]
		node.Files = detectPageFiles(rootDir, route)
		analyzerCtx.markPageScript(node.Files.JS)

		title, components := parsePageMetadata(rootDir, route, node.Files.JSON)
		if title == "" {
			title = globalTitle
		}
		node.Title = title
		node.UsingComponents = components
		node.Dependencies = extractJSDependencies(rootDir, node.Files.JS)
		node.APIUsage = extractPageAPIUsage(rootDir, route, node.Files.JS)
		node.APIUsage = append(node.APIUsage, extractIndirectAPIUsage(rootDir, route, node.Dependencies)...)
		sortPageAPIUsage(node.APIUsage)
		manifest.NavigationEdges = append(manifest.NavigationEdges, extractNavigationEdges(analyzerCtx, route, node.Files.JS, node.Files.WXML)...)
		manifest.Pages = append(manifest.Pages, *node)
	}

	declaredRoutes := make(map[string]bool, len(manifest.Pages))
	for _, page := range manifest.Pages {
		declaredRoutes[page.Route] = true
	}
	for i := range manifest.NavigationEdges {
		manifest.NavigationEdges[i].TargetExists = declaredRoutes[manifest.NavigationEdges[i].TargetPage]
	}

	manifest.OrphanPages = findOrphanPages(rootDir, declaredRoutes)
	manifest.SharedRouterHelpers = buildSharedRouterHelpers(manifest)
	sortManifest(manifest)
	manifest.Summary = buildSummary(manifest)

	return manifest, nil
}

func loadAppConfig(rootDir string) (*appConfig, string, error) {
	candidates := []string{
		filepath.Join(rootDir, "app.json"),
		filepath.Join(rootDir, "app-config.json"),
	}

	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		var app appConfig
		if err := json.Unmarshal(data, &app); err != nil {
			return nil, "", fmt.Errorf("解析配置文件失败 %s: %w", candidate, err)
		}
		if len(app.Window) == 0 && len(app.Global) > 0 {
			if window, ok := app.Global["window"].(map[string]interface{}); ok {
				app.Window = window
			}
		}
		if len(app.Pages) == 0 && len(app.SubPackages) == 0 && len(app.Subpackages) == 0 {
			continue
		}
		return &app, filepath.Base(candidate), nil
	}

	return nil, "", fmt.Errorf("未找到可用的 app.json 或 app-config.json")
}

func extractTabBar(tabBar map[string]interface{}) ([]TabBarItem, map[string]bool) {
	if len(tabBar) == 0 {
		return nil, map[string]bool{}
	}

	list, ok := tabBar["list"].([]interface{})
	if !ok {
		return nil, map[string]bool{}
	}

	items := make([]TabBarItem, 0, len(list))
	pageSet := make(map[string]bool)
	for _, item := range list {
		entry, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		pagePath := normalizeRoute(stringFromMap(entry, "pagePath"))
		if pagePath == "" {
			continue
		}
		pageSet[pagePath] = true
		items = append(items, TabBarItem{
			PagePath:         pagePath,
			Text:             stringFromMap(entry, "text"),
			IconPath:         normalizeAssetPath(stringFromMap(entry, "iconPath")),
			SelectedIconPath: normalizeAssetPath(stringFromMap(entry, "selectedIconPath")),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].PagePath < items[j].PagePath
	})
	return items, pageSet
}

func detectPageFiles(rootDir, route string) PageFiles {
	files := PageFiles{}
	if exists(rootDir, route+".js") {
		files.JS = route + ".js"
	}
	if exists(rootDir, route+".wxml") {
		files.WXML = route + ".wxml"
	}
	if exists(rootDir, route+".wxss") {
		files.WXSS = route + ".wxss"
	}
	if exists(rootDir, route+".json") {
		files.JSON = route + ".json"
	}
	return files
}

func parsePageMetadata(rootDir, route, jsonPath string) (string, []string) {
	if jsonPath == "" {
		return "", nil
	}
	data, err := os.ReadFile(filepath.Join(rootDir, filepath.FromSlash(jsonPath)))
	if err != nil {
		return "", nil
	}

	var pageConfig pageJSONConfig
	if err := json.Unmarshal(data, &pageConfig); err != nil {
		return "", nil
	}

	components := make([]string, 0, len(pageConfig.UsingComponents))
	for _, componentPath := range pageConfig.UsingComponents {
		components = append(components, normalizeComponentPath(route, componentPath))
	}
	return pageConfig.NavigationBarTitleText, dedupeAndSortStrings(components)
}

func extractPageAPIUsage(rootDir, route, jsPath string) []PageAPIUsage {
	if jsPath == "" {
		return nil
	}

	data, err := os.ReadFile(filepath.Join(rootDir, filepath.FromSlash(jsPath)))
	if err != nil {
		return nil
	}

	endpoints := scanner.ExtractAPIEndpoints(jsPath, data)
	results := make([]PageAPIUsage, 0, len(endpoints))
	for _, endpoint := range endpoints {
		if isInternalPageURL(rootDir, route, endpoint.RawURL) {
			continue
		}
		results = append(results, PageAPIUsage{
			Name:       endpoint.Name,
			Method:     endpoint.Method,
			RawURL:     endpoint.RawURL,
			FilePath:   endpoint.FilePath,
			LineNumber: endpoint.LineNumber,
			SourceRule: endpoint.SourceRule,
			SourceKind: "direct",
		})
	}
	sortPageAPIUsage(results)
	return results
}

func extractIndirectAPIUsage(rootDir, route string, dependencies []string) []PageAPIUsage {
	if len(dependencies) == 0 {
		return nil
	}

	results := make([]PageAPIUsage, 0)
	seen := make(map[string]bool)
	for _, dependency := range dependencies {
		data, err := os.ReadFile(filepath.Join(rootDir, filepath.FromSlash(dependency)))
		if err != nil {
			continue
		}
		endpoints := scanner.ExtractAPIEndpoints(dependency, data)
		for _, endpoint := range endpoints {
			if isInternalPageURL(rootDir, route, endpoint.RawURL) {
				continue
			}
			key := strings.Join([]string{dependency, endpoint.Method, endpoint.RawURL}, "|")
			if seen[key] {
				continue
			}
			seen[key] = true
			results = append(results, PageAPIUsage{
				Name:       endpoint.Name,
				Method:     endpoint.Method,
				RawURL:     endpoint.RawURL,
				FilePath:   endpoint.FilePath,
				LineNumber: endpoint.LineNumber,
				SourceRule: endpoint.SourceRule,
				SourceKind: "indirect",
				ViaModule:  dependency,
			})
		}
	}

	sortPageAPIUsage(results)
	return results
}

func sortPageAPIUsage(results []PageAPIUsage) {
	sort.Slice(results, func(i, j int) bool {
		if results[i].SourceKind != results[j].SourceKind {
			return results[i].SourceKind < results[j].SourceKind
		}
		if results[i].LineNumber != results[j].LineNumber {
			return results[i].LineNumber < results[j].LineNumber
		}
		if results[i].Method != results[j].Method {
			return results[i].Method < results[j].Method
		}
		if results[i].ViaModule != results[j].ViaModule {
			return results[i].ViaModule < results[j].ViaModule
		}
		return results[i].RawURL < results[j].RawURL
	})
}

func extractNavigationEdges(ctx *routeAnalyzerContext, route, jsPath, wxmlPath string) []NavigationEdge {
	results := make([]NavigationEdge, 0)
	jsEdgesByHandler := make(map[string][]NavigationEdge)
	usedHandlerKeys := make(map[string]bool)
	if jsPath != "" {
		for _, edge := range extractJSNavigationEdges(ctx.rootDir, route, jsPath) {
			if edge.HandlerName != "" {
				jsEdgesByHandler[edge.HandlerName] = append(jsEdgesByHandler[edge.HandlerName], edge)
				continue
			}
			results = append(results, edge)
		}
	}

	actions := make([]wxmlAction, 0)
	if wxmlPath != "" {
		data, err := os.ReadFile(filepath.Join(ctx.rootDir, filepath.FromSlash(wxmlPath)))
		if err == nil {
			wxmlEdges, consumed, extractedActions := extractWXMLNavigationEdges(route, wxmlPath, string(data), jsEdgesByHandler)
			results = append(results, wxmlEdges...)
			actions = extractedActions
			for _, key := range consumed {
				usedHandlerKeys[key] = true
			}
		}
	}

	if jsPath != "" {
		results = append(results, extractCallChainNavigationEdges(ctx, route, jsPath, actions)...)
		results = append(results, extractLifecycleNavigationEdges(ctx, route, jsPath)...)
	}

	for handlerName, edges := range jsEdgesByHandler {
		for _, edge := range edges {
			key := handlerEdgeKey(handlerName, edge)
			if usedHandlerKeys[key] {
				continue
			}
			results = append(results, edge)
		}
	}
	return dedupeEdges(results)
}

func extractJSNavigationEdges(rootDir, route, jsPath string) []NavigationEdge {
	data, err := os.ReadFile(filepath.Join(rootDir, filepath.FromSlash(jsPath)))
	if err != nil {
		return nil
	}

	text := string(data)
	handlers := extractJSHandlers(text)
	matches := jsNavigationCallStartPattern.FindAllStringSubmatchIndex(text, -1)
	results := make([]NavigationEdge, 0, len(matches))
	for _, match := range matches {
		if len(match) < 4 {
			continue
		}
		blockStart := strings.Index(text[match[0]:match[1]], "{")
		if blockStart < 0 {
			continue
		}
		blockStart += match[0]
		blockEnd := findMatchingBrace(text, blockStart)
		if blockEnd <= blockStart {
			continue
		}
		method := strings.TrimSpace(text[match[2]:match[3]])
		urlExpr := extractObjectPropertyExpression(text[blockStart:blockEnd+1], "url")
		target, rawTarget, dynamic, ok := resolveNavigationExpression(urlExpr, route)
		if !ok {
			continue
		}
		results = append(results, NavigationEdge{
			SourcePage:  route,
			TargetPage:  target,
			RawTarget:   rawTarget,
			Method:      method,
			SourceType:  "js",
			SourceFile:  jsPath,
			LineNumber:  lineNumberAtOffset(text, match[0]),
			HandlerName: findContainingHandler(handlers, match[0]),
			Dynamic:     dynamic,
		})
	}
	return results
}

func extractWXMLNavigationEdges(route, wxmlPath string, text string, jsEdgesByHandler map[string][]NavigationEdge) ([]NavigationEdge, []string, []wxmlAction) {
	results := make([]NavigationEdge, 0)
	consumed := make([]string, 0)

	for _, match := range wxmlNavigatorBlockPattern.FindAllStringSubmatchIndex(text, -1) {
		if len(match) < 6 {
			continue
		}
		attrs := text[match[2]:match[3]]
		inner := text[match[4]:match[5]]
		urlMatch := attrURLPattern.FindStringSubmatch(attrs)
		if len(urlMatch) < 2 {
			continue
		}
		rawTarget := strings.TrimSpace(urlMatch[1])
		target, ok := normalizeRouteReference(rawTarget, route)
		if !ok {
			continue
		}
		method := "navigateTo"
		if openTypeMatch := attrOpenTypePattern.FindStringSubmatch(attrs); len(openTypeMatch) > 1 {
			method = strings.TrimSpace(openTypeMatch[1])
		}
		results = append(results, NavigationEdge{
			SourcePage:   route,
			TargetPage:   target,
			RawTarget:    rawTarget,
			Method:       method,
			SourceType:   "wxml",
			SourceFile:   wxmlPath,
			LineNumber:   lineNumberAtOffset(text, match[0]),
			TriggerEvent: "tap",
			TriggerText:  compactText(inner),
		})
	}

	actions := append(extractWXMLActions(text, wxmlPath), extractWXMLSelfCloseActions(text, wxmlPath)...)
	actions = append(actions, extractWXMLLineActions(text, wxmlPath)...)
	for _, action := range actions {
		if action.HandlerName == "" {
			continue
		}
		candidates := jsEdgesByHandler[action.HandlerName]
		if len(candidates) == 0 {
			if action.RawTarget == "" {
				continue
			}
			target, ok := normalizeRouteReference(action.RawTarget, route)
			if !ok {
				continue
			}
			results = append(results, NavigationEdge{
				SourcePage:   route,
				TargetPage:   target,
				RawTarget:    action.RawTarget,
				Method:       "UNKNOWN",
				SourceType:   "wxml-event",
				SourceFile:   action.SourceFile,
				LineNumber:   action.LineNumber,
				HandlerName:  action.HandlerName,
				TriggerEvent: action.TriggerEvent,
				TriggerText:  action.TriggerText,
				Dynamic:      strings.Contains(action.RawTarget, "{{"),
			})
			continue
		}

		for _, candidate := range candidates {
			cloned := candidate
			cloned.SourceType = "js-handler"
			cloned.SourceFile = action.SourceFile
			cloned.LineNumber = action.LineNumber
			cloned.TriggerEvent = action.TriggerEvent
			cloned.TriggerText = action.TriggerText

			if action.RawTarget != "" && (candidate.Dynamic || strings.Contains(candidate.RawTarget, "dataset.")) {
				if target, ok := normalizeRouteReference(action.RawTarget, route); ok {
					cloned.TargetPage = target
					cloned.RawTarget = action.RawTarget
					cloned.Dynamic = strings.Contains(action.RawTarget, "{{")
				}
			}
			if cloned.TargetPage == "" && cloned.RawTarget == "" {
				continue
			}
			results = append(results, cloned)
			consumed = append(consumed, handlerEdgeKey(action.HandlerName, candidate))
		}
	}

	return results, dedupeAndSortStrings(consumed), actions
}

func extractJSDependencies(rootDir, jsPath string) []string {
	if jsPath == "" {
		return nil
	}

	visited := make(map[string]bool)
	results := make([]string, 0)
	var walk func(relPath string, depth int)
	walk = func(relPath string, depth int) {
		if depth > 2 || relPath == "" || visited[relPath] {
			return
		}
		visited[relPath] = true

		data, err := os.ReadFile(filepath.Join(rootDir, filepath.FromSlash(relPath)))
		if err != nil {
			return
		}
		text := string(data)
		imports := make([]string, 0)
		for _, match := range requirePattern.FindAllStringSubmatch(text, -1) {
			if len(match) > 1 {
				imports = append(imports, match[1])
			}
		}
		for _, match := range importPattern.FindAllStringSubmatch(text, -1) {
			if len(match) > 1 {
				imports = append(imports, match[1])
			}
		}

		for _, imp := range imports {
			resolved := resolveJSImport(rootDir, relPath, imp)
			if resolved == "" || resolved == jsPath {
				continue
			}
			results = append(results, resolved)
			walk(resolved, depth+1)
		}
	}

	walk(jsPath, 0)
	filtered := make([]string, 0, len(results))
	seen := make(map[string]bool)
	for _, dependency := range results {
		if dependency == jsPath || seen[dependency] {
			continue
		}
		seen[dependency] = true
		filtered = append(filtered, dependency)
	}
	sort.Strings(filtered)
	return filtered
}

func resolveJSImport(rootDir, fromJSPath, spec string) string {
	spec = strings.TrimSpace(spec)
	if spec == "" || strings.Contains(spec, "://") {
		return ""
	}
	if !strings.HasPrefix(spec, ".") && !strings.HasPrefix(spec, "/") {
		return ""
	}

	var base string
	if strings.HasPrefix(spec, "/") {
		base = normalizeRoute(spec)
	} else {
		base = normalizeRoute(path.Join(path.Dir(strings.TrimSuffix(fromJSPath, filepath.Ext(fromJSPath))), spec))
	}
	if base == "" {
		return ""
	}

	candidates := []string{
		base + ".js",
		path.Join(base, "index.js"),
	}
	for _, candidate := range candidates {
		if exists(rootDir, candidate) {
			return candidate
		}
	}
	return ""
}

func extractJSHandlers(text string) []jsHandler {
	patterns := []*regexp.Regexp{methodPatternFunc, methodPatternArrow, methodPatternShort}
	results := make([]jsHandler, 0)
	seen := make(map[string]bool)
	for _, pattern := range patterns {
		for _, match := range pattern.FindAllStringSubmatchIndex(text, -1) {
			if len(match) < 4 {
				continue
			}
			name := text[match[2]:match[3]]
			if isReservedJSName(name) {
				continue
			}
			openBraceOffset := strings.LastIndex(text[match[0]:match[1]], "{")
			if openBraceOffset < 0 {
				continue
			}
			openBrace := match[0] + openBraceOffset
			closeBrace := findMatchingBrace(text, openBrace)
			if closeBrace <= openBrace {
				continue
			}
			key := fmt.Sprintf("%s:%d:%d", name, openBrace, closeBrace)
			if seen[key] {
				continue
			}
			seen[key] = true
			results = append(results, jsHandler{
				Name:  name,
				Start: match[0],
				End:   closeBrace,
				Body:  text[openBrace : closeBrace+1],
			})
		}
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Start != results[j].Start {
			return results[i].Start < results[j].Start
		}
		return results[i].Name < results[j].Name
	})
	return results
}

func isReservedJSName(name string) bool {
	switch name {
	case "if", "for", "switch", "while", "catch", "function", "return", "else":
		return true
	default:
		return false
	}
}

func findContainingHandler(handlers []jsHandler, offset int) string {
	for _, handler := range handlers {
		if offset >= handler.Start && offset <= handler.End {
			return handler.Name
		}
	}
	return ""
}

func findMatchingBrace(text string, openBrace int) int {
	if openBrace < 0 || openBrace >= len(text) || text[openBrace] != '{' {
		return -1
	}

	depth := 0
	var quote byte
	inLineComment := false
	inBlockComment := false
	escaped := false

	for i := openBrace; i < len(text); i++ {
		ch := text[i]
		next := byte(0)
		if i+1 < len(text) {
			next = text[i+1]
		}

		if inLineComment {
			if ch == '\n' {
				inLineComment = false
			}
			continue
		}
		if inBlockComment {
			if ch == '*' && next == '/' {
				inBlockComment = false
				i++
			}
			continue
		}
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}

		if ch == '/' && next == '/' {
			inLineComment = true
			i++
			continue
		}
		if ch == '/' && next == '*' {
			inBlockComment = true
			i++
			continue
		}
		if ch == '\'' || ch == '"' || ch == '`' {
			quote = ch
			continue
		}
		if ch == '{' {
			depth++
			continue
		}
		if ch == '}' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func extractObjectPropertyExpression(objectText, property string) string {
	pattern := property + ":"
	idx := strings.Index(strings.ToLower(objectText), pattern)
	if idx < 0 {
		return ""
	}
	idx += len(pattern)
	for idx < len(objectText) && isWhitespace(objectText[idx]) {
		idx++
	}
	if idx >= len(objectText) {
		return ""
	}

	start := idx
	var quote byte
	parenDepth := 0
	bracketDepth := 0
	braceDepth := 0
	escaped := false

	for idx < len(objectText) {
		ch := objectText[idx]
		if quote != 0 {
			if escaped {
				escaped = false
				idx++
				continue
			}
			if ch == '\\' {
				escaped = true
				idx++
				continue
			}
			if ch == quote {
				quote = 0
			}
			idx++
			continue
		}

		switch ch {
		case '\'', '"', '`':
			quote = ch
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		case '{':
			braceDepth++
		case '}':
			if braceDepth == 0 {
				return strings.TrimSpace(strings.TrimSuffix(objectText[start:idx], ","))
			}
			braceDepth--
		case ',':
			if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 {
				return strings.TrimSpace(objectText[start:idx])
			}
		}
		idx++
	}

	return strings.TrimSpace(objectText[start:])
}

func resolveNavigationExpression(expr, currentRoute string) (string, string, bool, bool) {
	value := strings.TrimSpace(strings.TrimSuffix(expr, ","))
	if value == "" {
		return "", "", false, false
	}

	if len(value) >= 2 {
		if (value[0] == '\'' && value[len(value)-1] == '\'') || (value[0] == '"' && value[len(value)-1] == '"') {
			raw := value[1 : len(value)-1]
			target, ok := normalizeRouteReference(raw, currentRoute)
			return target, raw, false, ok
		}
		if value[0] == '`' && value[len(value)-1] == '`' {
			raw := value[1 : len(value)-1]
			dynamic := strings.Contains(raw, "${")
			staticPart := strings.Split(raw, "${")[0]
			if target, ok := normalizeRouteReference(staticPart, currentRoute); ok {
				return target, raw, dynamic, true
			}
			return "", raw, dynamic, raw != ""
		}
	}

	if literal := firstStringLiteral(value); literal != "" {
		target, ok := normalizeRouteReference(literal, currentRoute)
		return target, value, true, ok || value != ""
	}

	return "", value, true, true
}

func firstStringLiteral(expr string) string {
	match := jsURLLiteralPattern.FindStringSubmatch(expr)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func extractWXMLActions(text, sourceFile string) []wxmlAction {
	results := make([]wxmlAction, 0)
	for _, match := range wxmlActionBlockPattern.FindAllStringSubmatchIndex(text, -1) {
		if len(match) < 8 {
			continue
		}
		tag := text[match[2]:match[3]]
		if strings.EqualFold(tag, "navigator") {
			continue
		}
		attrs := text[match[4]:match[5]]
		inner := text[match[6]:match[7]]
		action, ok := buildWXMLAction(tag, attrs, inner, sourceFile, lineNumberAtOffset(text, match[0]))
		if ok {
			results = append(results, action)
		}
	}
	return results
}

func extractWXMLSelfCloseActions(text, sourceFile string) []wxmlAction {
	results := make([]wxmlAction, 0)
	for _, match := range wxmlActionSelfClosePattern.FindAllStringSubmatchIndex(text, -1) {
		if len(match) < 6 {
			continue
		}
		tag := text[match[2]:match[3]]
		if strings.EqualFold(tag, "navigator") {
			continue
		}
		attrs := text[match[4]:match[5]]
		action, ok := buildWXMLAction(tag, attrs, "", sourceFile, lineNumberAtOffset(text, match[0]))
		if ok {
			results = append(results, action)
		}
	}
	return results
}

func extractWXMLLineActions(text, sourceFile string) []wxmlAction {
	lines := strings.Split(text, "\n")
	results := make([]wxmlAction, 0)
	for index, line := range lines {
		if !attrEventPattern.MatchString(line) {
			continue
		}
		tagStart := strings.Index(line, "<")
		openEndRel := strings.Index(line[tagStart:], ">")
		if tagStart < 0 || openEndRel < 0 {
			continue
		}
		openEnd := tagStart + openEndRel
		tagChunk := line[tagStart+1 : openEnd]
		tagName := strings.Fields(tagChunk)
		if len(tagName) == 0 || strings.EqualFold(tagName[0], "navigator") {
			continue
		}
		action, ok := buildWXMLAction(tagName[0], tagChunk, line[openEnd+1:], sourceFile, index+1)
		if ok {
			results = append(results, action)
		}
	}
	return results
}

func buildWXMLAction(tag, attrs, inner, sourceFile string, lineNumber int) (wxmlAction, bool) {
	eventMatch := attrEventPattern.FindStringSubmatch(attrs)
	if len(eventMatch) < 3 {
		return wxmlAction{}, false
	}
	rawTarget := ""
	if dataMatch := attrDataTargetPattern.FindStringSubmatch(attrs); len(dataMatch) > 2 {
		rawTarget = strings.TrimSpace(dataMatch[2])
	}
	return wxmlAction{
		Tag:          strings.TrimSpace(tag),
		TriggerEvent: normalizeTriggerEvent(eventMatch[1]),
		HandlerName:  strings.TrimSpace(eventMatch[2]),
		RawTarget:    rawTarget,
		TriggerText:  compactText(inner),
		LineNumber:   lineNumber,
		SourceFile:   sourceFile,
	}, true
}

func normalizeTriggerEvent(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "capture-", "")
	value = strings.ReplaceAll(value, "bind:", "")
	value = strings.ReplaceAll(value, "catch:", "")
	value = strings.ReplaceAll(value, "bind", "")
	value = strings.ReplaceAll(value, "catch", "")
	return strings.Trim(value, ":")
}

func compactText(raw string) string {
	cleaned := regexp.MustCompile(`(?is)<[^>]+>`).ReplaceAllString(raw, " ")
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	return strings.TrimSpace(cleaned)
}

func handlerEdgeKey(handlerName string, edge NavigationEdge) string {
	return strings.Join([]string{
		handlerName,
		edge.Method,
		edge.RawTarget,
		edge.TargetPage,
		edge.SourceFile,
		fmt.Sprintf("%d", edge.LineNumber),
	}, "|")
}

func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\n' || ch == '\r' || ch == '\t'
}

func findOrphanPages(rootDir string, declaredRoutes map[string]bool) []string {
	results := make([]string, 0)
	seen := make(map[string]bool)

	_ = filepath.WalkDir(rootDir, func(pathValue string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".gwxapkg" {
				return fs.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(rootDir, pathValue)
		if err != nil {
			return nil
		}
		relPath = filepath.ToSlash(relPath)
		if shouldIgnoreGeneratedArtifact(relPath) || filepath.Ext(relPath) != ".js" {
			return nil
		}

		route := normalizeRoute(strings.TrimSuffix(relPath, filepath.Ext(relPath)))
		if route == "" || declaredRoutes[route] || seen[route] {
			return nil
		}

		content, readErr := os.ReadFile(pathValue)
		if readErr != nil || !pageScriptPattern.Match(content) {
			return nil
		}

		seen[route] = true
		results = append(results, route)
		return nil
	})

	sort.Strings(results)
	return results
}

func buildSummary(manifest *RouteManifest) RouteSummary {
	summary := RouteSummary{
		TotalPages:               len(manifest.Pages),
		TabBarPages:              len(manifest.TabBar),
		NavigationEdgeCount:      len(manifest.NavigationEdges),
		SharedRouterHelperCount:  len(manifest.SharedRouterHelpers),
		ExternalMiniProgramCount: len(manifest.ExternalMiniPrograms),
		OrphanPageCount:          len(manifest.OrphanPages),
	}

	componentSet := make(map[string]struct{})
	for _, page := range manifest.Pages {
		if page.PackageType == "subpackage" {
			summary.SubPackagePages++
		} else {
			summary.MainPages++
		}
		if len(page.APIUsage) > 0 {
			summary.PagesWithAPI++
		}
		for _, endpoint := range page.APIUsage {
			summary.APIEndpointCount++
			if endpoint.SourceKind == "indirect" {
				summary.IndirectAPIEndpointCount++
			}
		}
		for _, component := range page.UsingComponents {
			componentSet[component] = struct{}{}
		}
	}
	for _, edge := range manifest.NavigationEdges {
		if edge.Dynamic {
			summary.DynamicNavigationEdgeCount++
		}
		if len(edge.CallChain) > 1 {
			summary.CallChainEdgeCount++
		}
	}

	summary.ReferencedComponents = len(componentSet)
	return summary
}

func sortManifest(manifest *RouteManifest) {
	sort.Slice(manifest.Pages, func(i, j int) bool {
		return manifest.Pages[i].Route < manifest.Pages[j].Route
	})
	sort.Slice(manifest.SubPackages, func(i, j int) bool {
		return manifest.SubPackages[i].Root < manifest.SubPackages[j].Root
	})
	sort.Slice(manifest.SharedRouterHelpers, func(i, j int) bool {
		left := manifest.SharedRouterHelpers[i]
		right := manifest.SharedRouterHelpers[j]
		if left.FilePath != right.FilePath {
			return left.FilePath < right.FilePath
		}
		return left.FunctionName < right.FunctionName
	})
	sort.Slice(manifest.NavigationEdges, func(i, j int) bool {
		left := manifest.NavigationEdges[i]
		right := manifest.NavigationEdges[j]
		if left.SourcePage != right.SourcePage {
			return left.SourcePage < right.SourcePage
		}
		if left.LineNumber != right.LineNumber {
			return left.LineNumber < right.LineNumber
		}
		if left.Method != right.Method {
			return left.Method < right.Method
		}
		if left.HandlerName != right.HandlerName {
			return left.HandlerName < right.HandlerName
		}
		if left.TriggerText != right.TriggerText {
			return left.TriggerText < right.TriggerText
		}
		if len(left.CallChain) != len(right.CallChain) {
			return len(left.CallChain) > len(right.CallChain)
		}
		return left.TargetPage < right.TargetPage
	})
}

func dedupeEdges(edges []NavigationEdge) []NavigationEdge {
	if len(edges) == 0 {
		return nil
	}

	results := make([]NavigationEdge, 0, len(edges))
	indexByKey := make(map[string]int)
	for _, edge := range edges {
		key := edgeIdentityKey(edge)
		if index, ok := indexByKey[key]; ok {
			results[index] = mergeNavigationEdge(results[index], edge)
			continue
		}
		indexByKey[key] = len(results)
		results = append(results, edge)
	}
	return pruneFallbackNavigationEdges(results)
}

func edgeIdentityKey(edge NavigationEdge) string {
	return strings.Join([]string{
		edge.SourcePage,
		edge.TargetPage,
		edge.RawTarget,
		edge.Method,
		edge.HandlerName,
		edge.TriggerEvent,
		edge.TriggerText,
		fmt.Sprintf("%t", edge.Dynamic),
		fmt.Sprintf("%d", edge.LineNumber),
	}, "|")
}

func mergeNavigationEdge(current, candidate NavigationEdge) NavigationEdge {
	if navigationEdgeScore(candidate) > navigationEdgeScore(current) {
		current, candidate = candidate, current
	}

	if current.SourceFile == "" {
		current.SourceFile = candidate.SourceFile
	}
	if current.LineNumber == 0 {
		current.LineNumber = candidate.LineNumber
	}
	if current.SourceType == "" {
		current.SourceType = candidate.SourceType
	}
	if current.HandlerName == "" {
		current.HandlerName = candidate.HandlerName
	}
	if current.TriggerEvent == "" {
		current.TriggerEvent = candidate.TriggerEvent
	}
	if current.TriggerText == "" {
		current.TriggerText = candidate.TriggerText
	}
	if current.TargetPage == "" {
		current.TargetPage = candidate.TargetPage
	}
	if current.RawTarget == "" {
		current.RawTarget = candidate.RawTarget
	}
	if len(current.CallChain) == 0 {
		current.CallChain = candidate.CallChain
	}
	current.TargetExists = current.TargetExists || candidate.TargetExists
	current.Dynamic = current.Dynamic || candidate.Dynamic
	return current
}

func navigationEdgeScore(edge NavigationEdge) int {
	score := 0
	score += len(edge.CallChain) * 10
	if edge.SourceType == "shared-router" {
		score += 5
	}
	if edge.TriggerText != "" {
		score += 3
	}
	if edge.TriggerEvent != "" {
		score += 2
	}
	if edge.HandlerName != "" {
		score++
	}
	return score
}

func pruneFallbackNavigationEdges(edges []NavigationEdge) []NavigationEdge {
	richKeys := make(map[string]bool)
	for _, edge := range edges {
		if edge.Method != "" && edge.Method != "UNKNOWN" {
			richKeys[fallbackNavigationKey(edge)] = true
		}
	}

	filtered := make([]NavigationEdge, 0, len(edges))
	for _, edge := range edges {
		if edge.Method == "UNKNOWN" && edge.SourceType == "wxml-event" && richKeys[fallbackNavigationKey(edge)] {
			continue
		}
		filtered = append(filtered, edge)
	}
	return filtered
}

func fallbackNavigationKey(edge NavigationEdge) string {
	return strings.Join([]string{
		edge.SourcePage,
		edge.TargetPage,
		edge.RawTarget,
		edge.HandlerName,
		edge.TriggerEvent,
		edge.TriggerText,
		fmt.Sprintf("%d", edge.LineNumber),
	}, "|")
}

func normalizeRouteReference(rawTarget, currentRoute string) (string, bool) {
	candidate := strings.TrimSpace(rawTarget)
	if candidate == "" {
		return "", false
	}

	if strings.Contains(candidate, "://") || strings.HasPrefix(candidate, "//") || strings.Contains(candidate, "{{") {
		return "", false
	}

	if idx := strings.IndexAny(candidate, "?#"); idx >= 0 {
		candidate = candidate[:idx]
	}
	candidate = normalizeAssetPath(candidate)
	if candidate == "" {
		return "", false
	}

	if strings.HasPrefix(candidate, "/") {
		return normalizeRoute(candidate), true
	}

	baseDir := path.Dir(currentRoute)
	return normalizeRoute(path.Join(baseDir, candidate)), true
}

func isInternalPageURL(rootDir, currentRoute, rawTarget string) bool {
	target, ok := normalizeRouteReference(rawTarget, currentRoute)
	if !ok || target == "" {
		return false
	}
	if strings.HasPrefix(target, "api/") || versionAPIPathPattern.MatchString(target) {
		return false
	}
	files := detectPageFiles(rootDir, target)
	return files.JS != "" || files.WXML != "" || files.JSON != ""
}

func normalizeComponentPath(currentRoute, componentPath string) string {
	candidate := strings.TrimSpace(componentPath)
	if candidate == "" {
		return ""
	}
	if strings.Contains(candidate, "://") {
		return candidate
	}
	if strings.HasPrefix(candidate, "/") {
		return normalizeRoute(candidate)
	}
	baseDir := path.Dir(currentRoute)
	return normalizeRoute(path.Join(baseDir, candidate))
}

func normalizeRoute(value string) string {
	candidate := normalizeAssetPath(value)
	candidate = strings.TrimPrefix(candidate, "./")
	candidate = strings.TrimPrefix(candidate, "/")
	candidate = path.Clean(candidate)
	if candidate == "." || candidate == "" {
		return ""
	}
	for _, ext := range []string{".js", ".wxml", ".wxss", ".json", ".html"} {
		candidate = strings.TrimSuffix(candidate, ext)
	}
	return strings.TrimPrefix(candidate, "/")
}

func normalizeAssetPath(value string) string {
	candidate := strings.TrimSpace(value)
	candidate = strings.ReplaceAll(candidate, "\\", "/")
	return strings.TrimSpace(candidate)
}

func joinRoute(root, page string) string {
	root = normalizeRoute(root)
	page = normalizeRoute(page)
	switch {
	case root == "":
		return page
	case page == "":
		return root
	default:
		return normalizeRoute(path.Join(root, page))
	}
}

func exists(rootDir, relPath string) bool {
	_, err := os.Stat(filepath.Join(rootDir, filepath.FromSlash(relPath)))
	return err == nil
}

func lineNumberAtOffset(text string, offset int) int {
	if offset <= 0 {
		return 1
	}
	count := 1
	for i := 0; i < len(text) && i < offset; i++ {
		if text[i] == '\n' {
			count++
		}
	}
	return count
}

func dedupeAndSortStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(values))
	results := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		results = append(results, value)
	}
	sort.Strings(results)
	return results
}

func stringFromMap(data map[string]interface{}, key string) string {
	if len(data) == 0 {
		return ""
	}
	value, ok := data[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return ""
	}
}

func shouldIgnoreGeneratedArtifact(relPath string) bool {
	if relPath == ".gwxapkg" || strings.HasPrefix(relPath, ".gwxapkg/") {
		return true
	}

	name := path.Base(relPath)
	switch name {
	case "sensitive_report.html",
		"sensitive_report.xlsx",
		"api_collection.postman_collection.json",
		"route_manifest.json",
		"route_map.md",
		"route_map.mmd":
		return true
	default:
		return false
	}
}
