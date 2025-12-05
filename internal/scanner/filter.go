package scanner

import (
	"regexp"
	"strings"
)

// 黑名单
var (
	// 文件名黑名单
	fileNameBlacklist = map[string]bool{
		"index.weapp": true, "index.html": true, "index.wxss": true,
		"index.wxml": true, "main.html": true, "Date.now": true,
		"index.js": true, "app.js": true, "common.js": true,
		"document.dispatchEvent": true, "Math.random": true,
		"Object.keys": true, "Array.from": true,
		"document.getElementById": true, "document.querySelector": true,
		"window.location": true, "console.log": true,
		"Promise.resolve": true, "setTimeout": true,
		"index.weapp.wxss": true, "index.weapp.wxml": true,
		"area.wxss": true, "area.wxml": true, "result.wxss": true,
		"result.wxml": true, "page-list.wxss": true, "page-list.wxml": true,
		"page-question.wxss": true, "page-question.wxml": true,
		"page-banner.wxss": true, "page-banner.wxml": true,
		"page-empty.wxss": true, "page-empty.wxml": true,
		"project-list.wxss": true, "project-list.wxml": true,
		"fuse-component.wxss": true, "fuse-component.wxml": true,
		"banner.weapp": true, "confirm.wxss": true, "confirm.wxml": true,
		"auth.html": true, "doc.html": true, "bearPayPlugin.html": true,
		"canvas.html": true, "canvas2c.html": true, "json2canvas.html": true,
		"scope.userLocation": true,
	}

	// 文件扩展名模式
	fileExtPattern = regexp.MustCompile(`\.(weapp|html|js|wxss|wxml|css|json|xml|png|jpg|jpeg|gif|svg|ico|woff|woff2|ttf|eot)$`)

	// 常见 TLD
	validTLDs = map[string]bool{
		"com": true, "cn": true, "net": true, "org": true, "io": true,
		"gov": true, "edu": true, "mil": true, "co": true, "uk": true,
		"us": true, "jp": true, "kr": true, "de": true, "fr": true,
		"ru": true, "au": true, "ca": true, "in": true, "br": true,
		"mx": true, "it": true, "es": true, "nl": true, "pl": true,
		"se": true, "no": true, "dk": true, "fi": true, "be": true,
		"at": true, "ch": true, "cz": true, "gr": true, "pt": true,
		// 中国常见域名后缀
		"myhuaweicloud": true, "aliyuncs": true, "myscrm": true,
		"myunke": true, "iwofang": true, "weixin": true, "qq": true,
		"dingtalk": true, "feishu": true,
	}
)

// SensitiveFilter 误报过滤器
type SensitiveFilter struct {
	blacklist map[string]bool
}

// NewFilter 创建过滤器
func NewFilter() *SensitiveFilter {
	return &SensitiveFilter{
		blacklist: fileNameBlacklist,
	}
}

// ShouldSkip 判断是否应该跳过（误报）
func (f *SensitiveFilter) ShouldSkip(ruleID, content, context string) bool {
	// 去除空白字符
	content = strings.TrimSpace(content)
	if content == "" {
		return true
	}

	// 黑名单过滤
	if f.blacklist[content] {
		return true
	}

	// 域名规则的特殊过滤
	if ruleID == "domain" {
		return f.isDomainFalsePositive(content, context)
	}

	// Path 规则的特殊过滤
	if ruleID == "path" {
		return f.isPathFalsePositive(content)
	}

	return false
}

// isDomainFalsePositive 判断域名规则的误报
func (f *SensitiveFilter) isDomainFalsePositive(content, context string) bool {
	// 1. 文件扩展名过滤
	if fileExtPattern.MatchString(content) {
		return true
	}

	// 2. 没有有效 TLD
	if !hasValidTLD(content) {
		return true
	}

	// 3. 在 JS API 调用中
	if isJavaScriptAPI(content, context) {
		return true
	}

	// 4. 单个词（不包含点）
	if !strings.Contains(content, ".") {
		return true
	}

	return false
}

// isPathFalsePositive 判断路径规则的误报
func (f *SensitiveFilter) isPathFalsePositive(content string) bool {
	// 过滤引号
	content = strings.Trim(content, "\"'")

	// 过滤太短的路径
	if len(content) < 5 {
		return true
	}

	return false
}

// hasValidTLD 检查是否有有效的顶级域名
func hasValidTLD(domain string) bool {
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return false
	}

	tld := strings.ToLower(parts[len(parts)-1])
	return validTLDs[tld]
}

// isJavaScriptAPI 判断是否是 JS API
func isJavaScriptAPI(content, context string) bool {
	// 常见的 JS API 模式
	jsPatterns := []string{
		"Date\\.",
		"Math\\.",
		"Object\\.",
		"Array\\.",
		"document\\.",
		"window\\.",
		"console\\.",
		"JSON\\.",
		"Promise\\.",
		"Number\\.",
		"String\\.",
		"Boolean\\.",
	}

	for _, pattern := range jsPatterns {
		matched, _ := regexp.MatchString(pattern, context)
		if matched {
			return true
		}
	}

	return false
}
