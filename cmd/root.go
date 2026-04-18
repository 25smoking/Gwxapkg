package cmd

import (
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"

	. "github.com/25smoking/Gwxapkg/internal/cmd"
	. "github.com/25smoking/Gwxapkg/internal/config"
	"github.com/25smoking/Gwxapkg/internal/key"
	packmeta "github.com/25smoking/Gwxapkg/internal/pack"
	"github.com/25smoking/Gwxapkg/internal/reporter"
	"github.com/25smoking/Gwxapkg/internal/restore"
	"github.com/25smoking/Gwxapkg/internal/ui"
	"github.com/25smoking/Gwxapkg/internal/util"
)

func Execute(appID, input, outputDir, fileExt string, restoreDir bool, pretty bool, noClean bool, save bool, sensitive bool, workspace bool) {
	// 确定输出目录
	if outputDir == "" {
		outputDir = DetermineOutputDir(input, appID)
	}
	expandedOutputDir, err := util.ExpandHomePath(outputDir)
	if err != nil {
		ui.Warning("展开输出目录失败，继续使用原路径: %v", err)
	} else {
		outputDir = expandedOutputDir
	}

	// 存储配置
	configManager := NewSharedConfigManager()
	configManager.Set("appID", appID)
	configManager.Set("input", input)
	configManager.Set("outputDir", outputDir)
	configManager.Set("fileExt", fileExt)
	configManager.Set("restoreDir", restoreDir)
	configManager.Set("pretty", pretty)
	configManager.Set("noClean", noClean)
	configManager.Set("save", save)
	configManager.Set("sensitive", sensitive)
	configManager.Set("workspace", workspace)

	inputFiles := ParseInput(input, fileExt)

	if len(inputFiles) == 0 {
		ui.Warning("未找到任何文件")
		return
	}

	//  如果启用敏感信息扫描，初始化scanner
	if sensitive {
		if err := key.InitRules(); err != nil {
			ui.Warning("初始化扫描规则失败: %v", err)
			sensitive = false
		} else {
			key.InitCollector(appID)
		}
	}

	// 显示步骤信息
	ui.Step(1, 2, "解包 wxapkg 文件...")

	// 创建进度条
	bar := ui.NewProgressBar(len(inputFiles), "解包中")

	var wg sync.WaitGroup
	var errCount int32

	for _, inputFile := range inputFiles {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			err := ProcessFile(file, outputDir, appID, save, workspace)
			if err != nil {
				atomic.AddInt32(&errCount, 1)
			}
			bar.Add(1)
		}(inputFile)
	}
	wg.Wait()

	// 显示解包结果
	if errCount > 0 {
		ui.Warning("解包完成，%d 个文件处理失败", errCount)
	}

	// 为保留原始包内容的场景生成 manifest，方便后续精确回包
	if workspace || !restoreDir || noClean {
		if err := packmeta.WritePackageManifest(outputDir, appID, GetWxapkgManager()); err != nil {
			ui.Warning("写入回包 manifest 失败: %v", err)
		}
	}

	// 还原工程目录结构
	ui.Step(2, 2, "还原工程结构...")
	restore.ProjectStructure(outputDir, restoreDir)

	// 输出结果目录
	fmt.Println()
	ui.Success("输出目录: %s", filepath.Clean(outputDir))

	// 如果启用了敏感信息扫描，生成Excel报告
	if sensitive {
		collector := key.GetCollector()
		if collector != nil {
			collector.SetTotalFiles(len(inputFiles))
			report := collector.GenerateReport()

			// 生成 Excel 报告
			excelReporter := reporter.NewExcelReporter()
			reportPath := filepath.Join(outputDir, "sensitive_report.xlsx")
			if err := excelReporter.Generate(report, reportPath); err != nil {
				ui.Warning("生成扫描报告失败: %v", err)
			} else {
				ui.Success("敏感信息报告: %s", reportPath)
				ui.Info("   - 总匹配数: %d", report.Summary.TotalMatches)
				ui.Info("   - 去重后: %d", report.Summary.UniqueMatches)
				ui.Info("   - 高风险: %d | 中风险: %d | 低风险: %d",
					report.Summary.HighRisk, report.Summary.MediumRisk, report.Summary.LowRisk)
			}

			// 清理收集器
			key.ResetCollector()
		}
	}
}
