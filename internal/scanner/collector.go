package scanner

import (
	"fmt"
	"sync"
	"time"
)

// DataCollector 数据收集器
type DataCollector struct {
	mu         sync.Mutex
	items      []SensitiveItem
	dedup      map[string]*DedupInfo
	categories map[string]*CategoryData
	appID      string
	totalFiles int
	filter     *SensitiveFilter
}

// NewCollector 创建收集器
func NewCollector(appID string) *DataCollector {
	return &DataCollector{
		items:      make([]SensitiveItem, 0),
		dedup:      make(map[string]*DedupInfo),
		categories: make(map[string]*CategoryData),
		appID:      appID,
		filter:     NewFilter(),
	}
}

// Add 添加匹配项
func (c *DataCollector) Add(item SensitiveItem) {
	// 过滤误报
	if c.filter.ShouldSkip(item.RuleID, item.Content, item.Context) {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 去重 key
	key := fmt.Sprintf("%s:%s", item.RuleID, item.Content)

	if dedup, exists := c.dedup[key]; exists {
		// 已存在，只添加位置
		dedup.Locations = append(dedup.Locations, LocationInfo{
			FilePath:   item.FilePath,
			LineNumber: item.LineNumber,
		})
		dedup.Count++
	} else {
		// 新项目
		c.dedup[key] = &DedupInfo{
			FirstItem: item,
			Locations: []LocationInfo{{
				FilePath:   item.FilePath,
				LineNumber: item.LineNumber,
			}},
			Count: 1,
		}
		c.items = append(c.items, item)

		// 添加到分类
		c.addToCategory(item)
	}
}

// addToCategory 添加到分类
func (c *DataCollector) addToCategory(item SensitiveItem) {
	category := item.Category
	if category == "" {
		category = getCategoryKey(item.RuleID)
	}

	if c.categories[category] == nil {
		c.categories[category] = &CategoryData{
			Name:  getCategoryName(category),
			Items: make(map[string][]LocationInfo),
		}
	}

	cat := c.categories[category]
	cat.Count++

	if cat.Items[item.Content] == nil {
		cat.UniqueCount++
	}

	cat.Items[item.Content] = append(cat.Items[item.Content], LocationInfo{
		FilePath:   item.FilePath,
		LineNumber: item.LineNumber,
	})
}

// SetTotalFiles 设置总文件数
func (c *DataCollector) SetTotalFiles(count int) {
	c.totalFiles = count
}

// GenerateReport 生成报告
func (c *DataCollector) GenerateReport() *ScanReport {
	c.mu.Lock()
	defer c.mu.Unlock()

	summary := c.generateSummary()

	return &ScanReport{
		AppID:      c.appID,
		ScanTime:   time.Now().Format("2006-01-02 15:04:05"),
		TotalFiles: c.totalFiles,
		Categories: c.categories,
		Items:      c.items,
		Summary:    summary,
	}
}

// generateSummary 生成摘要
func (c *DataCollector) generateSummary() ReportSummary {
	summary := ReportSummary{
		TotalMatches:  0,
		UniqueMatches: len(c.dedup),
		CategoryStats: make(map[string]int),
	}

	// 统计总匹配数和分类
	for _, dedup := range c.dedup {
		summary.TotalMatches += dedup.Count
	}

	for category, data := range c.categories {
		summary.CategoryStats[category] = data.UniqueCount
	}

	// 统计风险等级
	for _, item := range c.items {
		switch item.Confidence {
		case "high":
			summary.HighRisk++
		case "medium":
			summary.MediumRisk++
		default:
			summary.LowRisk++
		}
	}

	return summary
}

// getCategoryKey 根据 rule_id 获取分类 key
func getCategoryKey(ruleID string) string {
	categoryMap := map[string]string{
		"path":         "path",
		"url":          "url",
		"api_endpoint": "url",
		"domain":       "domain",
		
		// 密码和密钥
		"password_generic":   "password",
		"admin_password":     "password",
		"root_password":      "password",
		"default_password":   "password",
		"test_password":      "password",
		"ftp_password":       "password",
		"smtp_password":      "password",
		"ldap_password":      "password",
		"vpn_password":       "password",
		"wifi_password":      "password",
		"encryption_password": "password",
		"username_password":  "password",
		
		// API Keys
		"api_key_generic":     "api_key",
		"aws_access_key_id":   "api_key",
		"aliyun_access_key":   "api_key",
		"tencent_secret_id":   "api_key",
		"google_api_key":      "api_key",
		"github_pat":          "api_key",
		"gitlab_pat":          "api_key",
		
		// Secrets
		"secret_key_generic":  "secret",
		"aws_secret_access_key": "secret",
		"client_secret":       "secret",
		"app_secret":          "secret",
		"wechat_secret":       "secret",
		
		// Tokens
		"bearer_token":        "token",
		"api_token":           "token",
		"auth_token":          "token",
		"session_token":       "token",
		"access_token":        "token",
		
		// 数据库
		"jdbc_mysql":          "database",
		"jdbc_postgresql":     "database",
		"jdbc_oracle":         "database",
		"mongodb_connection":  "database",
		"redis_connection":    "database",
		"db_username":         "database",
		"db_password":         "database",
		"db_host":             "database",
		
		// 联系信息
		"phone_cn":            "contact",
		"email":               "contact",
		"id_card_cn":          "contact",
		
		// 其他
		"ipv4":                "network",
		"internal_ip":         "network",
		"uuid":                "other",
		"wechat_appid":        "wechat",
		"wechat_corpid":       "wechat",
	}

	if cat, ok := categoryMap[ruleID]; ok {
		return cat
	}
	return "other"
}

// getCategoryName 获取分类中文名
func getCategoryName(category string) string {
	names := map[string]string{
		"path":     "路径",
		"url":      "URL",
		"domain":   "域名",
		"password": "账号密码",
		"api_key":  "API密钥",
		"secret":   "密钥",
		"token":    "令牌",
		"database": "数据库",
		"contact":  "联系信息",
		"network":  "网络信息",
		"wechat":   "微信",
		"other":    "其他",
	}

	if name, ok := names[category]; ok {
		return name
	}
	return category
}
