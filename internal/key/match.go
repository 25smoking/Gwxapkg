package key

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/25smoking/Gwxapkg/internal/scanner"
)

var (
	rulesInstance *Rules
	once          sync.Once
	jsonMutex     sync.Mutex
	globalCollector *scanner.DataCollector
	collectorMutex  sync.Mutex
)

func getRulesInstance() (*Rules, error) {
	var err error
	once.Do(func() {
		rulesInstance, err = ReadRuleFile()
	})
	return rulesInstance, err
}

func MatchRules(input string) error {
	rules, err := getRulesInstance()
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	for _, rule := range rules.Rules {
		if rule.Enabled {
			re, err := regexp.Compile(rule.Pattern)
			if err != nil {
				return fmt.Errorf("failed to compile regex for rule %s: %v", rule.Id, err)
			}
			matches := re.FindAllStringSubmatch(input, -1)
			for _, match := range matches {
				if len(match) > 0 {
					if strings.TrimSpace(match[0]) == "" {
						continue
					}
					err := appendToJSON(rule.Id, match[0])
					if err != nil {
						return fmt.Errorf("failed to append to JSON: %v", err)
					}
				}
			}
		}
	}

	return nil
}

func appendToJSON(ruleId, matchedContent string) error {
	jsonMutex.Lock()
	defer jsonMutex.Unlock()

	file, err := os.OpenFile("sensitive_data.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open JSON file: %v", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Printf("failed to close JSON file: %v", err)
		}
	}(file)

	record := map[string]string{
		"rule_id": ruleId,
		"content": matchedContent,
	}

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(record); err != nil {
		return fmt.Errorf("failed to write to JSON file: %v", err)
	}

	return nil
}

// InitCollector 初始化全局收集器
func InitCollector(appID string) {
	collectorMutex.Lock()
	defer collectorMutex.Unlock()
	globalCollector = scanner.NewCollector(appID)
}

// GetCollector 获取全局收集器
func GetCollector() *scanner.DataCollector {
	collectorMutex.Lock()
	defer collectorMutex.Unlock()
	return globalCollector
}

// ResetCollector 重置收集器
func ResetCollector() {
	collectorMutex.Lock()
	defer collectorMutex.Unlock()
	globalCollector = nil
}

// InitRules 初始化规则（预编译）
func InitRules() error {
	rules, err := ReadRuleFile()
	if err != nil {
		return fmt.Errorf("读取规则文件失败: %w", err)
	}

	compiledRules := make([]*scanner.CompiledRule, 0)
	for _, rule := range rules.Rules {
		if !rule.Enabled {
			continue
		}

		pattern, e := regexp.Compile(rule.Pattern)
		if e != nil {
			fmt.Printf("警告: 规则 %s 编译失败: %v\n", rule.Id, e)
			continue
		}

		compiledRules = append(compiledRules, &scanner.CompiledRule{
			ID:         rule.Id,
			Pattern:    pattern,
			Category:   getCategoryKey(rule.Id),
			Confidence: getConfidence(rule.Id),
		})
	}
	
	scanner.CompiledRules = compiledRules
	return nil
}

// getCategoryKey 根据 rule_id 获取分类 key
func getCategoryKey(ruleID string) string {
	categoryMap := map[string]string{
		"path":         "path",
		"url":          "url",
		"api_endpoint": "url",
		"domain":       "domain",
		
		"password_generic":    "password",
		"admin_password":      "password",
		"root_password":       "password",
		"db_password":         "password",
		
		"api_key_generic":     "api_key",
		"aws_access_key_id":   "api_key",
		"aliyun_access_key":   "api_key",
		"tencent_secret_id":   "api_key",
		"google_api_key":      "api_key",
		
		"secret_key_generic":  "secret",
		"wechat_secret":       "secret",
		
		"bearer_token":        "token",
		"api_token":           "token",
		
		"jdbc_mysql":          "database",
		"mongodb_connection":  "database",
		
		"phone_cn":            "contact",
		"email":               "contact",
		
		"wechat_appid":        "wechat",
	}

	if cat, ok := categoryMap[ruleID]; ok {
		return cat
	}
	return "other"
}

// getConfidence 获取规则可信度
func getConfidence(ruleID string) string {	
	highConfidence := map[string]bool{
		"aws_access_key_id":   true,
		"private_key_rsa":     true,
		"github_pat":          true,
	}

	lowConfidence := map[string]bool{
		"domain":      true,
		"path":        true,
		"ipv4":        true,
		"uuid":        true,
	}

	if highConfidence[ruleID] {
		return "high"
	}
	if lowConfidence[ruleID] {
		return "low"
	}
	return "medium"
}
