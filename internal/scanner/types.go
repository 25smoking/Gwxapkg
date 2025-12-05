package scanner

// SensitiveItem 敏感数据项
type SensitiveItem struct {
	RuleID      string `json:"rule_id"`
	RuleName    string `json:"rule_name"`
	Category    string `json:"category"`
	Content     string `json:"content"`
	FilePath    string `json:"file_path"`
	LineNumber  int    `json:"line_number"`
	Context     string `json:"context"`     // 完整行内容
	Confidence  string `json:"confidence"`  // high/medium/low
	Timestamp   string `json:"timestamp"`
}

// LocationInfo 位置信息
type LocationInfo struct {
	FilePath   string `json:"file_path"`
	LineNumber int    `json:"line_number"`
}

// CategoryData 分类数据
type CategoryData struct {
	Name        string                     `json:"name"`
	Count       int                        `json:"count"`
	UniqueCount int                        `json:"unique_count"`
	Items       map[string][]LocationInfo  `json:"items"` // content -> locations
}

// ScanReport 扫描报告
type ScanReport struct {
	AppID      string                    `json:"app_id"`
	ScanTime   string                    `json:"scan_time"`
	TotalFiles int                       `json:"total_files"`
	Categories map[string]*CategoryData  `json:"categories"`
	Items      []SensitiveItem           `json:"items"`
	Summary    ReportSummary             `json:"summary"`
}

// ReportSummary 报告摘要
type ReportSummary struct {
	TotalMatches   int            `json:"total_matches"`
	UniqueMatches  int            `json:"unique_matches"`
	HighRisk       int            `json:"high_risk"`
	MediumRisk     int            `json:"medium_risk"`
	LowRisk        int            `json:"low_risk"`
	CategoryStats  map[string]int `json:"category_stats"`
}

// DedupInfo 去重信息
type DedupInfo struct {
	FirstItem  SensitiveItem
	Locations  []LocationInfo
	Count      int
}
