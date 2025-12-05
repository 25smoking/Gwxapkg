package scanner

import (
	"bufio"
	"regexp"
	"strings"
	"time"
)

// CompiledRule 编译后的规则
type CompiledRule struct {
	ID         string
	Pattern    *regexp.Regexp
	Category   string
	Confidence string
}

// CompiledRules 全局编译后的规则 (由 key 包设置)
var CompiledRules []*CompiledRule

// ScanFile 扫描单个文件
func ScanFile(filePath string, content []byte, collector *DataCollector) error {
	// 转换为字符串
	text := string(content)

	// 按行扫描以获取行号
	scanner := bufio.NewScanner(strings.NewReader(text))
	lineNumber := 1

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			lineNumber++
			continue
		}

		// 使用所有规则扫描这一行
		for _, rule := range CompiledRules {
			matches := rule.Pattern.FindAllString(line, -1)
			for _, match := range matches {
				if strings.TrimSpace(match) == "" {
					continue
				}

				item := SensitiveItem{
					RuleID:     rule.ID,
					RuleName:   getRuleName(rule.ID),
					Category:   rule.Category,
					Content:    match,
					FilePath:   filePath,
					LineNumber: lineNumber,
					Context:    line,
					Confidence: rule.Confidence,
					Timestamp:  time.Now().Format("2006-01-02 15:04:05"),
				}

				collector.Add(item)
			}
		}

		lineNumber++
	}

	return scanner.Err()
}

// getRuleName 获取规则中文名
func getRuleName(ruleID string) string {
	names := map[string]string{
		"email":         "邮箱",
		"phone_cn":      "手机号",
		"id_card_cn":    "身份证",
		"ipv4":          "IP地址",
		"path":          "路径",
		"url":           "URL",
		"api_endpoint":  "API端点",
		"domain":        "域名",
		
		"password_generic":    "密码",
		"admin_password":      "管理员密码",
		"root_password":       "Root密码",
		
		"api_key_generic":     "API密钥",
		"aws_access_key_id":   "AWS访问密钥",
		"aliyun_access_key":   "阿里云密钥",
		"tencent_secret_id":   "腾讯云密钥",
		"google_api_key":      "Google API密钥",
		
		"private_key_rsa":     "RSA私钥",
		"private_key_dsa":     "DSA私钥",
		"private_key_ec":      "EC私钥",
		
		"wechat_appid":        "微信AppID",
		"wechat_secret":       "微信Secret",
		
		"jdbc_mysql":          "MySQL连接",
		"mongodb_connection":  "MongoDB连接",
		"redis_connection":    "Redis连接",
	}

	if name, ok := names[ruleID]; ok {
		return name
	}
	return ruleID
}
