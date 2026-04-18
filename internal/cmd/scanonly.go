package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/25smoking/Gwxapkg/internal/key"
	"github.com/25smoking/Gwxapkg/internal/reporter"
	"github.com/25smoking/Gwxapkg/internal/scanner"
	"github.com/25smoking/Gwxapkg/internal/ui"
)

// ScanOnly 对已解包目录执行独立敏感信息扫描，生成报告
func ScanOnly(dir string, appID string, format string, outputDir string) {
	if _, err := os.Stat(dir); err != nil {
		ui.Error("目录不存在: %s", dir)
		return
	}

	// 如果未指定 AppID，用目录名
	if appID == "" {
		appID = filepath.Base(dir)
	}

	ui.Info("初始化扫描规则...")
	if err := key.InitRules(); err != nil {
		ui.Error("初始化规则失败: %v", err)
		return
	}
	key.InitCollector(appID)

	// 遍历目录，扫描所有文本文件
	ui.Step(1, 2, "扫描目录: %s", dir)
	collector := key.GetCollector()

	var fileCount int
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		// 只扫描文本类文件
		if !isTextFile(ext) {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		fileCount++
		relPath, _ := filepath.Rel(dir, path)
		_ = scanner.ScanFile(relPath, content, collector)
		return nil
	})
	if err != nil {
		ui.Warning("遍历目录出错: %v", err)
	}

	collector.SetTotalFiles(fileCount)
	report := collector.GenerateReport()
	key.ResetCollector()

	// 确定输出路径
	if outputDir == "" {
		outputDir = dir
	}

	ui.Step(2, 2, "生成报告...")

	// 生成报告（支持 excel / html / both）
	format = strings.ToLower(format)
	if format == "" {
		format = "both"
	}

	generated := 0
	if format == "excel" || format == "both" {
		path := filepath.Join(outputDir, "sensitive_report.xlsx")
		er := reporter.NewExcelReporter()
		if err := er.Generate(report, path); err != nil {
			ui.Warning("生成 Excel 报告失败: %v", err)
		} else {
			ui.Success("Excel 报告: %s", path)
			generated++
		}
	}
	if format == "html" || format == "both" {
		path := filepath.Join(outputDir, "sensitive_report.html")
		hr := reporter.NewHTMLReporter()
		if err := hr.Generate(report, path); err != nil {
			ui.Warning("生成 HTML 报告失败: %v", err)
		} else {
			ui.Success("HTML 报告: %s", path)
			generated++
		}
	}

	if generated == 0 {
		ui.Warning("未生成任何报告，请检查 -format 参数（excel/html/both）")
		return
	}

	fmt.Println()
	ui.Info("   - 扫描文件数: %d", fileCount)
	ui.Info("   - 总匹配数:   %d", report.Summary.TotalMatches)
	ui.Info("   - 去重后:     %d", report.Summary.UniqueMatches)
	ui.Info("   - 高风险: %d | 中风险: %d | 低风险: %d",
		report.Summary.HighRisk, report.Summary.MediumRisk, report.Summary.LowRisk)
}

// isTextFile 判断是否为需要扫描的文本文件
func isTextFile(ext string) bool {
	textExts := map[string]bool{
		".js": true, ".ts": true, ".json": true,
		".wxml": true, ".wxss": true, ".wxs": true,
		".html": true, ".css": true, ".xml": true,
		".txt": true, ".md": true, ".yaml": true, ".yml": true,
		".env": true, ".config": true, ".conf": true,
		".sh": true, ".bat": true, ".ps1": true,
		".go": true, ".py": true, ".rb": true, ".php": true,
		"": true, // 无扩展名文件也扫描
	}
	return textExts[ext]
}
