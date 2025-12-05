package reporter

import (
	"fmt"
	"sort"

	"github.com/25smoking/Gwxapkg/internal/scanner"
	"github.com/xuri/excelize/v2"
)

// ExcelReporter Excel 报告生成器
type ExcelReporter struct {
	file *excelize.File
}

// NewExcelReporter 创建 Excel 报告生成器
func NewExcelReporter() *ExcelReporter {
	return &ExcelReporter{
		file: excelize.NewFile(),
	}
}

// Generate 生成报告
func (r *ExcelReporter) Generate(report *scanner.ScanReport, filename string) error {
	// 1. 创建概览页
	if err := r.createOverviewSheet(report); err != nil {
		return fmt.Errorf("创建概览页失败: %w", err)
	}

	// 2. 创建分类页 (按分类排序)
	var categories []string
	for cat := range report.Categories {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	for _, category := range categories {
		data := report.Categories[category]
		if err := r.createCategorySheet(category, data); err != nil {
			return fmt.Errorf("创建分类页 %s 失败: %w", category, err)
		}
	}

	// 3. 应用样式
	r.applyStyles()

	// 4. 删除默认 Sheet
	r.file.DeleteSheet("Sheet1")

	// 5. 保存文件
	if err := r.file.SaveAs(filename); err != nil {
		return fmt.Errorf("保存文件失败: %w", err)
	}

	return nil
}

// createOverviewSheet 创建概览页
func (r *ExcelReporter) createOverviewSheet(report *scanner.ScanReport) error {
	sheet := "概览"
	index, err := r.file.NewSheet(sheet)
	if err != nil {
		return err
	}
	r.file.SetActiveSheet(index)

	// 标题
	r.file.SetCellValue(sheet, "A1", "敏感信息扫描报告")
	r.file.MergeCell(sheet, "A1", "D1")

	// 基本信息
	r.file.SetCellValue(sheet, "A3", "App ID:")
	r.file.SetCellValue(sheet, "B3", report.AppID)
	r.file.SetCellValue(sheet, "A4", "扫描时间:")
	r.file.SetCellValue(sheet, "B4", report.ScanTime)
	r.file.SetCellValue(sheet, "A5", "扫描文件数:")
	r.file.SetCellValue(sheet, "B5", report.TotalFiles)

	// 统计摘要
	r.file.SetCellValue(sheet, "A7", "统计摘要")
	r.file.MergeCell(sheet, "A7", "D7")
	
	r.file.SetCellValue(sheet, "A8", "总匹配数:")
	r.file.SetCellValue(sheet, "B8", report.Summary.TotalMatches)
	r.file.SetCellValue(sheet, "A9", "去重后:")
	r.file.SetCellValue(sheet, "B9", report.Summary.UniqueMatches)
	r.file.SetCellValue(sheet, "A10", "高风险:")
	r.file.SetCellValue(sheet, "B10", report.Summary.HighRisk)
	r.file.SetCellValue(sheet, "A11", "中风险:")
	r.file.SetCellValue(sheet, "B11", report.Summary.MediumRisk)
	r.file.SetCellValue(sheet, "A12", "低风险:")
	r.file.SetCellValue(sheet, "B12", report.Summary.LowRisk)

	// 分类统计表头
	r.file.SetCellValue(sheet, "A14", "分类统计")
	r.file.MergeCell(sheet, "A14", "C14")
	
	r.file.SetCellValue(sheet, "A15", "分类")
	r.file.SetCellValue(sheet, "B15", "数量")
	r.file.SetCellValue(sheet, "C15", "占比")

	// 分类统计数据
	row := 16
	totalUnique := report.Summary.UniqueMatches
	for category, count := range report.Summary.CategoryStats {
		categoryName := report.Categories[category].Name
		r.file.SetCellValue(sheet, fmt.Sprintf("A%d", row), categoryName)
		r.file.SetCellValue(sheet, fmt.Sprintf("B%d", row), count)
		
		percentage := float64(count) / float64(totalUnique) * 100
		r.file.SetCellValue(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("%.1f%%", percentage))
		row++
	}

	// 设置列宽
	r.file.SetColWidth(sheet, "A", "A", 18)
	r.file.SetColWidth(sheet, "B", "D", 15)

	return nil
}

// createCategorySheet 创建分类页
func (r *ExcelReporter) createCategorySheet(category string, data *scanner.CategoryData) error {
	sheetName := data.Name
	
	// Excel sheet 名称不能超过 31 字符
	if len(sheetName) > 31 {
		sheetName = sheetName[:31]
	}
	
	_, err := r.file.NewSheet(sheetName)
	if err != nil {
		return err
	}

	// 表头
	headers := []string{"序号", "内容", "出现次数", "文件路径", "行号"}
	for i, header := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		r.file.SetCellValue(sheetName, cell, header)
	}

	// 数据 (按内容排序)
	var contents []string
	for content := range data.Items {
		contents = append(contents, content)
	}
	sort.Strings(contents)

	row := 2
	idx := 1
	for _, content := range contents {
		locations := data.Items[content]

		r.file.SetCellValue(sheetName, fmt.Sprintf("A%d", row), idx)
		r.file.SetCellValue(sheetName, fmt.Sprintf("B%d", row), content)
		r.file.SetCellValue(sheetName, fmt.Sprintf("C%d", row), len(locations))

		if len(locations) > 0 {
			r.file.SetCellValue(sheetName, fmt.Sprintf("D%d", row), locations[0].FilePath)
			r.file.SetCellValue(sheetName, fmt.Sprintf("E%d", row), locations[0].LineNumber)
		}

		// 如果有多个位置，在后续行显示
		for i := 1; i < len(locations); i++ {
			row++
			r.file.SetCellValue(sheetName, fmt.Sprintf("D%d", row), locations[i].FilePath)
			r.file.SetCellValue(sheetName, fmt.Sprintf("E%d", row), locations[i].LineNumber)
		}

		row++
		idx++
	}

	// 设置列宽
	r.file.SetColWidth(sheetName, "A", "A", 8)
	r.file.SetColWidth(sheetName, "B", "B", 50)
	r.file.SetColWidth(sheetName, "C", "C", 12)
	r.file.SetColWidth(sheetName, "D", "D", 40)
	r.file.SetColWidth(sheetName, "E", "E", 10)

	return nil
}

// applyStyles 应用样式
func (r *ExcelReporter) applyStyles() {
	// 创建标题样式
	titleStyle, _ := r.file.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
			Size: 16,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})

	// 创建表头样式
	headerStyle, _ := r.file.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Color: "FFFFFF",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"4472C4"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})

	// 创建小标题样式
	subtitleStyle, _ := r.file.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
			Size: 12,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"E7E6E6"},
			Pattern: 1,
		},
	})

	// 应用样式到各个 sheet
	sheets := r.file.GetSheetList()
	for _, sheet := range sheets {
		if sheet == "概览" {
			// 标题行
			r.file.SetCellStyle(sheet, "A1", "D1", titleStyle)
			// 小标题
			r.file.SetCellStyle(sheet, "A7", "D7", subtitleStyle)
			r.file.SetCellStyle(sheet, "A14", "C14", subtitleStyle)
			// 表头行
			r.file.SetCellStyle(sheet, "A15", "C15", headerStyle)
		} else {
			// 分类页表头行
			r.file.SetCellStyle(sheet, "A1", "E1", headerStyle)
		}
	}
}
