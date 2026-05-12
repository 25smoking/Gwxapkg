package scanner

import (
	"fmt"
	"slices"
	"sync"
	"time"
)

// DataCollector 数据收集器
type DataCollector struct {
	mu              sync.Mutex
	items           []SensitiveItem
	dedup           map[string]*DedupInfo
	categories      map[string]*CategoryData
	apiEndpoints    []APIEndpoint
	apiDedup        map[string]struct{}
	obfuscatedFiles []ObfuscatedFile
	obfuscatedIndex map[string]int
	appID           string
	totalFiles      int
	filter          *SensitiveFilter
}

// NewCollector 创建收集器
func NewCollector(appID string) *DataCollector {
	return &DataCollector{
		items:           make([]SensitiveItem, 0),
		dedup:           make(map[string]*DedupInfo),
		categories:      make(map[string]*CategoryData),
		apiEndpoints:    make([]APIEndpoint, 0),
		apiDedup:        make(map[string]struct{}),
		obfuscatedFiles: make([]ObfuscatedFile, 0),
		obfuscatedIndex: make(map[string]int),
		appID:           appID,
		filter:          NewFilter(),
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

// AddAPIEndpoint 添加接口信息
func (c *DataCollector) AddAPIEndpoint(endpoint APIEndpoint) {
	if endpoint.RawURL == "" {
		return
	}

	if endpoint.Method == "" {
		endpoint.Method = "UNKNOWN"
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	key := fmt.Sprintf("%s:%s", endpoint.Method, endpoint.RawURL)
	if _, exists := c.apiDedup[key]; exists {
		return
	}

	c.apiDedup[key] = struct{}{}
	c.apiEndpoints = append(c.apiEndpoints, endpoint)
}

// AddObfuscatedFile 添加混淆文件信息
func (c *DataCollector) AddObfuscatedFile(file ObfuscatedFile) {
	if file.FilePath == "" {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if idx, exists := c.obfuscatedIndex[file.FilePath]; exists {
		current := c.obfuscatedFiles[idx]
		current.Score = max(current.Score, file.Score)
		current.Techniques = mergeTechniques(current.Techniques, file.Techniques)
		current.Status = mergeObfuscatedStatus(current.Status, file.Status)
		if current.Tag == "" {
			current.Tag = file.Tag
		}
		c.obfuscatedFiles[idx] = current
		return
	}

	c.obfuscatedIndex[file.FilePath] = len(c.obfuscatedFiles)
	c.obfuscatedFiles = append(c.obfuscatedFiles, file)
}

// addToCategory 添加到分类
func (c *DataCollector) addToCategory(item SensitiveItem) {
	category := item.Category
	if category == "" {
		category = GetCategoryKey(item.RuleID)
	}

	if c.categories[category] == nil {
		c.categories[category] = &CategoryData{
			Name:  GetCategoryName(category),
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

// RewriteFilePaths 在后处理重命名模块后同步报告中的文件路径。
func (c *DataCollector) RewriteFilePaths(pathMap map[string]string) {
	if len(pathMap) == 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	rename := func(value string) string {
		if next, ok := pathMap[value]; ok {
			return next
		}
		return value
	}

	for i := range c.items {
		c.items[i].FilePath = rename(c.items[i].FilePath)
	}
	for _, dedup := range c.dedup {
		dedup.FirstItem.FilePath = rename(dedup.FirstItem.FilePath)
		for i := range dedup.Locations {
			dedup.Locations[i].FilePath = rename(dedup.Locations[i].FilePath)
		}
	}
	for _, category := range c.categories {
		for content, locations := range category.Items {
			for i := range locations {
				locations[i].FilePath = rename(locations[i].FilePath)
			}
			category.Items[content] = locations
		}
	}
	for i := range c.apiEndpoints {
		c.apiEndpoints[i].FilePath = rename(c.apiEndpoints[i].FilePath)
	}

	c.obfuscatedIndex = make(map[string]int, len(c.obfuscatedFiles))
	for i := range c.obfuscatedFiles {
		c.obfuscatedFiles[i].FilePath = rename(c.obfuscatedFiles[i].FilePath)
		c.obfuscatedIndex[c.obfuscatedFiles[i].FilePath] = i
	}
}

// GenerateReport 生成报告
func (c *DataCollector) GenerateReport() *ScanReport {
	c.mu.Lock()
	defer c.mu.Unlock()

	summary := c.generateSummary()

	return &ScanReport{
		AppID:           c.appID,
		ScanTime:        time.Now().Format("2006-01-02 15:04:05"),
		TotalFiles:      c.totalFiles,
		Categories:      c.categories,
		Items:           c.items,
		APIEndpoints:    slices.Clone(c.apiEndpoints),
		ObfuscatedFiles: cloneObfuscatedFiles(c.obfuscatedFiles),
		Summary:         summary,
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

func mergeTechniques(left, right []string) []string {
	if len(left) == 0 {
		return slices.Clone(right)
	}

	seen := make(map[string]struct{}, len(left)+len(right))
	merged := make([]string, 0, len(left)+len(right))
	for _, technique := range append(slices.Clone(left), right...) {
		if technique == "" {
			continue
		}
		if _, exists := seen[technique]; exists {
			continue
		}
		seen[technique] = struct{}{}
		merged = append(merged, technique)
	}
	return merged
}

func mergeObfuscatedStatus(left, right string) string {
	order := map[string]int{
		"":         0,
		"flagged":  1,
		"partial":  2,
		"restored": 3,
	}

	if order[right] > order[left] {
		return right
	}
	return left
}

func cloneObfuscatedFiles(files []ObfuscatedFile) []ObfuscatedFile {
	result := make([]ObfuscatedFile, 0, len(files))
	for _, file := range files {
		cloned := file
		cloned.Techniques = slices.Clone(file.Techniques)
		result = append(result, cloned)
	}
	return result
}
