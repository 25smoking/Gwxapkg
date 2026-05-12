package semantic

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	burpAPILinkJSONFileName = "burp_api_link.json"
	burpAPILinkMDFileName   = "burp_api_link.md"
)

// BurpAPILinkReport 描述 Burp 原始请求与源码 API 的关联结果。
type BurpAPILinkReport struct {
	GeneratedAt string             `json:"generated_at"`
	Request     ParsedBurpRequest  `json:"request"`
	Matches     []BurpAPILinkMatch `json:"matches"`
}

// ParsedBurpRequest 是 Burp HTTP 原始包的本地解析结果。
type ParsedBurpRequest struct {
	Method         string            `json:"method"`
	Path           string            `json:"path"`
	Host           string            `json:"host,omitempty"`
	ContentType    string            `json:"content_type,omitempty"`
	Params         map[string]string `json:"params,omitempty"`
	ControllerName string            `json:"controller_name,omitempty"`
	MethodsName    string            `json:"methods_name,omitempty"`
}

// BurpAPILinkMatch 表示一个候选源码 API。
type BurpAPILinkMatch struct {
	Confidence     string   `json:"confidence"`
	Reason         string   `json:"reason"`
	Score          int      `json:"score"`
	FunctionName   string   `json:"function_name"`
	ControllerName string   `json:"controller_name"`
	MethodsName    string   `json:"methods_name"`
	HTTPMethod     string   `json:"http_method,omitempty"`
	FilePath       string   `json:"file_path"`
	ParamFields    []string `json:"param_fields,omitempty"`
	CallSites      []string `json:"call_sites,omitempty"`
}

// LinkBurpRequest 将 Burp 原始请求关联到 api_map 中的源码 API。
func LinkBurpRequest(rootDir string, rawRequest string) (*BurpAPILinkReport, error) {
	rootAbs, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("解析输出目录失败: %w", err)
	}
	apiMap, err := readAPIMap(rootAbs)
	if err != nil {
		return nil, err
	}
	parsed := ParseBurpRequest(rawRequest)
	report := &BurpAPILinkReport{
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		Request:     parsed,
		Matches:     matchBurpRequestToAPI(parsed, apiMap),
	}
	if err := writeBurpAPILink(rootAbs, report); err != nil {
		return nil, err
	}
	return report, nil
}

func readAPIMap(rootDir string) (*APIMapReport, error) {
	data, err := os.ReadFile(filepath.Join(rootDir, reportDirName, "api_map.json"))
	if err != nil {
		return nil, fmt.Errorf("读取 api_map.json 失败: %w", err)
	}
	var report APIMapReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("解析 api_map.json 失败: %w", err)
	}
	return &report, nil
}

// ParseBurpRequest 只做本地解析，不发送请求。
func ParseBurpRequest(rawRequest string) ParsedBurpRequest {
	normalized := strings.ReplaceAll(rawRequest, "\r\n", "\n")
	head, body, _ := strings.Cut(normalized, "\n\n")
	lines := strings.Split(head, "\n")
	result := ParsedBurpRequest{Params: make(map[string]string)}
	if len(lines) == 0 {
		return result
	}
	parts := strings.Fields(lines[0])
	if len(parts) >= 2 {
		result.Method = strings.ToUpper(parts[0])
		result.Path = parts[1]
		if parsedURL, err := url.Parse(parts[1]); err == nil {
			result.Path = parsedURL.Path
			addQueryParams(result.Params, parsedURL.Query())
		}
	}
	for _, line := range lines[1:] {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(strings.ToLower(key))
		value = strings.TrimSpace(value)
		switch key {
		case "host":
			result.Host = value
		case "content-type":
			result.ContentType = value
		}
	}
	parseBodyParams(result.Params, result.ContentType, body)
	result.ControllerName = firstParamValue(result.Params, "controllerName", "controller")
	result.MethodsName = firstParamValue(result.Params, "methodsName", "methodName", "method")
	return result
}

func addQueryParams(params map[string]string, values url.Values) {
	for key, value := range values {
		if len(value) > 0 {
			params[key] = value[0]
		}
	}
}

func parseBodyParams(params map[string]string, contentType, body string) {
	body = strings.TrimSpace(body)
	if body == "" {
		return
	}
	if strings.Contains(strings.ToLower(contentType), "application/json") || strings.HasPrefix(body, "{") {
		var decoded interface{}
		if err := json.Unmarshal([]byte(body), &decoded); err == nil {
			flattenJSONParams(params, "", decoded)
			return
		}
	}
	if values, err := url.ParseQuery(body); err == nil {
		addQueryParams(params, values)
	}
}

func flattenJSONParams(params map[string]string, prefix string, value interface{}) {
	switch typed := value.(type) {
	case map[string]interface{}:
		for key, item := range typed {
			nextKey := key
			if prefix != "" {
				nextKey = prefix + "." + key
			}
			flattenJSONParams(params, nextKey, item)
		}
	case []interface{}:
		params[prefix] = fmt.Sprintf("%d items", len(typed))
	default:
		if prefix != "" {
			params[prefix] = fmt.Sprint(typed)
		}
	}
}

func firstParamValue(params map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(params[key]); value != "" {
			return value
		}
	}
	return ""
}

func matchBurpRequestToAPI(request ParsedBurpRequest, apiMap *APIMapReport) []BurpAPILinkMatch {
	matches := make([]BurpAPILinkMatch, 0)
	for _, endpoint := range apiMap.Endpoints {
		match := scoreBurpEndpoint(request, endpoint)
		if match.Score <= 0 {
			continue
		}
		matches = append(matches, match)
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score != matches[j].Score {
			return matches[i].Score > matches[j].Score
		}
		return matches[i].FunctionName < matches[j].FunctionName
	})
	if len(matches) > 10 {
		matches = matches[:10]
	}
	return matches
}

func scoreBurpEndpoint(request ParsedBurpRequest, endpoint APIEndpointEntry) BurpAPILinkMatch {
	match := BurpAPILinkMatch{
		FunctionName:   endpoint.FunctionName,
		ControllerName: endpoint.ControllerName,
		MethodsName:    endpoint.MethodsName,
		HTTPMethod:     endpoint.HTTPMethod,
		FilePath:       endpoint.FilePath,
		ParamFields:    endpoint.ParamFields,
		CallSites:      callSiteStrings(endpoint.CallSites),
	}
	if request.ControllerName != "" && request.MethodsName != "" &&
		strings.EqualFold(request.ControllerName, endpoint.ControllerName) &&
		strings.EqualFold(request.MethodsName, endpoint.MethodsName) {
		match.Confidence = ASTConfidenceHigh
		match.Reason = "controllerName/methodsName 精确匹配"
		match.Score = 100
		return match
	}

	paramOverlap := overlapCount(request.Params, endpoint.ParamFields)
	methodMatch := endpoint.HTTPMethod == "" || request.Method == "" || strings.EqualFold(request.Method, endpoint.HTTPMethod)
	pathText := strings.ToLower(request.Path)
	nameHit := strings.Contains(pathText, strings.ToLower(endpoint.FunctionName)) ||
		strings.Contains(pathText, strings.ToLower(endpoint.ControllerName)) ||
		strings.Contains(pathText, strings.ToLower(endpoint.MethodsName))

	if methodMatch && (nameHit || paramOverlap >= max(1, min(2, len(endpoint.ParamFields)))) {
		match.Confidence = ASTConfidenceMedium
		match.Reason = "HTTP method/path/参数字段组合匹配"
		match.Score = 50 + paramOverlap*10
		if nameHit {
			match.Score += 15
		}
		return match
	}
	if paramOverlap > 0 || rawContainsEndpointName(request, endpoint) {
		match.Confidence = ASTConfidenceLow
		match.Reason = "参数字段或函数名弱匹配"
		match.Score = 10 + paramOverlap*5
		return match
	}
	return match
}

func overlapCount(params map[string]string, fields []string) int {
	count := 0
	for _, field := range fields {
		if _, ok := params[field]; ok {
			count++
		}
	}
	return count
}

func rawContainsEndpointName(request ParsedBurpRequest, endpoint APIEndpointEntry) bool {
	text := strings.ToLower(request.Path + " " + strings.Join(mapKeysString(request.Params), " "))
	return strings.Contains(text, strings.ToLower(endpoint.FunctionName)) ||
		strings.Contains(text, strings.ToLower(endpoint.MethodsName))
}

func mapKeysString(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func callSiteStrings(callSites []APICallSite) []string {
	results := make([]string, 0, len(callSites))
	for _, call := range callSites {
		results = append(results, fmt.Sprintf("%s:%d", call.FilePath, call.LineNumber))
	}
	return results
}

func writeBurpAPILink(rootDir string, report *BurpAPILinkReport) error {
	reportDir := filepath.Join(rootDir, reportDirName)
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(reportDir, burpAPILinkJSONFileName), data, 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(reportDir, burpAPILinkMDFileName), []byte(buildBurpAPILinkMarkdown(report)), 0644)
}

func buildBurpAPILinkMarkdown(report *BurpAPILinkReport) string {
	var builder strings.Builder
	builder.WriteString("# Burp 请求源码 API 关联\n\n")
	builder.WriteString(fmt.Sprintf("- 请求: `%s %s`\n", report.Request.Method, report.Request.Path))
	if report.Request.Host != "" {
		builder.WriteString(fmt.Sprintf("- Host: `%s`\n", report.Request.Host))
	}
	if len(report.Matches) == 0 {
		builder.WriteString("\n未匹配到源码 API。\n")
		return builder.String()
	}
	builder.WriteString("\n| 置信度 | 函数 | Controller | Method | 文件 | 原因 |\n")
	builder.WriteString("|--------|------|------------|--------|------|------|\n")
	for _, match := range report.Matches {
		builder.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` | `%s` | %s |\n",
			match.Confidence, match.FunctionName, match.ControllerName, match.MethodsName, match.FilePath, match.Reason))
	}
	return builder.String()
}
