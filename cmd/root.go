package cmd

import (
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/25smoking/Gwxapkg/internal/analyzer"
	. "github.com/25smoking/Gwxapkg/internal/cmd"
	. "github.com/25smoking/Gwxapkg/internal/config"
	"github.com/25smoking/Gwxapkg/internal/key"
	packmeta "github.com/25smoking/Gwxapkg/internal/pack"
	"github.com/25smoking/Gwxapkg/internal/reporter"
	"github.com/25smoking/Gwxapkg/internal/restore"
	"github.com/25smoking/Gwxapkg/internal/semantic"
	"github.com/25smoking/Gwxapkg/internal/ui"
	"github.com/25smoking/Gwxapkg/internal/util"
)

func Execute(appID, input, outputDir, fileExt string, restoreDir bool, pretty bool, noClean bool, save bool, sensitive bool, postman bool, workspace bool) {
	ExecuteWithOptions(appID, input, outputDir, fileExt, restoreDir, pretty, noClean, save, sensitive, postman, workspace, semantic.DefaultRewriteOptions())
}

func ExecuteWithOptions(appID, input, outputDir, fileExt string, restoreDir bool, pretty bool, noClean bool, save bool, sensitive bool, postman bool, workspace bool, rewriteOptions semantic.RewriteOptions) {
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
	configManager.Set("postman", postman)
	configManager.Set("workspace", workspace)

	inputFiles := ParseInput(input, fileExt)

	if len(inputFiles) == 0 {
		ui.Warning("未找到任何文件")
		return
	}

	// 如果需要敏感扫描或 Postman 导出，初始化规则与收集器
	if sensitive || postman {
		if err := key.InitRules(); err != nil {
			ui.Warning("初始化扫描规则失败: %v", err)
			sensitive = false
			postman = false
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
	errChan := make(chan error, len(inputFiles))

	for _, inputFile := range inputFiles {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			err := ProcessFile(file, outputDir, appID, save, workspace)
			if err != nil {
				atomic.AddInt32(&errCount, 1)
				errChan <- err
			}
			bar.Add(1)
		}(inputFile)
	}
	wg.Wait()
	close(errChan)

	for err := range errChan {
		ui.Error("%v", err)
	}

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

	if restoreDir {
		printASTRenameNotice(rewriteOptions.ASTRename)
		semanticReport, err := semantic.RewriteProjectWithOptions(outputDir, rewriteOptions)
		if err != nil {
			ui.Warning("源码级语义反混淆失败: %v", err)
		} else {
			if collector := key.GetCollector(); collector != nil {
				collector.RewriteFilePaths(semanticReport.PathMap)
			}
			if semanticReport.RenamedCount > 0 || semanticReport.SourceMapRecovered > 0 {
				ui.Success("源码语义映射: %s", filepath.Join(outputDir, ".gwxapkg", "semantic_module_map.json"))
				ui.Info("   - 语义重命名: %d | require 重写: %d | SourceMap 源码: %d",
					semanticReport.RenamedCount,
					semanticReport.RewrittenRequireCount,
					semanticReport.SourceMapRecovered,
				)
			}
			if semanticReport.APIEndpointCount > 0 {
				ui.Success("API 地图: %s", filepath.Join(outputDir, ".gwxapkg", "api_map.md"))
				ui.Info("   - API 函数: %d | 细拆模块: %d",
					semanticReport.APIEndpointCount,
					semanticReport.APISplitCount,
				)
				ui.Success("API 调用链: %s", filepath.Join(outputDir, ".gwxapkg", "api_call_chain.md"))
				ui.Success("API 伪代码: %s", filepath.Join(outputDir, ".gwxapkg", "api_pseudo.md"))
			}
			if semanticReport.ASTRenamedCount > 0 {
				ui.Success("AST 重命名报告: %s", filepath.Join(outputDir, ".gwxapkg", "ast_rename_map.json"))
				ui.Info("   - AST 重命名: %d | 文件数: %d",
					semanticReport.ASTRenamedCount,
					semanticReport.ASTRenamedFiles,
				)
			}
		}
	}

	// 输出结果目录
	fmt.Println()
	ui.Success("输出目录: %s", filepath.Clean(outputDir))

	collector := key.GetCollector()
	if collector != nil {
		collector.SetTotalFiles(len(inputFiles))
		report := collector.GenerateReport()

		if sensitive {
			excelReporter := reporter.NewExcelReporter()
			excelPath := filepath.Join(outputDir, "sensitive_report.xlsx")
			if err := excelReporter.Generate(report, excelPath); err != nil {
				ui.Warning("生成 Excel 报告失败: %v", err)
			} else {
				ui.Success("Excel 报告: %s", excelPath)
			}

			htmlReporter := reporter.NewHTMLReporter()
			htmlPath := filepath.Join(outputDir, "sensitive_report.html")
			if err := htmlReporter.Generate(report, htmlPath); err != nil {
				ui.Warning("生成 HTML 报告失败: %v", err)
			} else {
				ui.Success("HTML 报告: %s", htmlPath)
			}
		}

		if postman {
			postmanReporter := reporter.NewPostmanReporter()
			postmanPath := filepath.Join(outputDir, "api_collection.postman_collection.json")
			if err := postmanReporter.Generate(report, postmanPath); err != nil {
				ui.Warning("生成 Postman Collection 失败: %v", err)
			} else {
				ui.Success("Postman Collection: %s", postmanPath)
			}
		}

		if sensitive || postman {
			ui.Info("   - 接口数: %d", len(report.APIEndpoints))
			ui.Info("   - 混淆文件: %d", len(report.ObfuscatedFiles))
		}
		if sensitive {
			ui.Info("   - 总匹配数: %d", report.Summary.TotalMatches)
			ui.Info("   - 去重后: %d", report.Summary.UniqueMatches)
			ui.Info("   - 高风险: %d | 中风险: %d | 低风险: %d",
				report.Summary.HighRisk, report.Summary.MediumRisk, report.Summary.LowRisk)
		}

		key.ResetCollector()
	}

	if restoreDir {
		routeManifest, routeErr := analyzer.AnalyzeMiniProgram(outputDir, appID)
		if routeErr != nil {
			ui.Warning("生成页面与路由地图失败: %v", routeErr)
			return
		}

		routeReporter := reporter.NewRouteReporter()
		artifacts, err := routeReporter.Generate(routeManifest, outputDir)
		if err != nil {
			ui.Warning("写入页面与路由地图失败: %v", err)
			return
		}

		ui.Success("页面路由清单: %s", artifacts.ManifestPath)
		ui.Success("页面路由说明: %s", artifacts.MarkdownPath)
		ui.Success("页面路由图: %s", artifacts.MermaidPath)
		ui.Info("   - 页面数: %d | 跳转边: %d | 调用链边: %d | 共享助手: %d | TabBar: %d",
			routeManifest.Summary.TotalPages,
			routeManifest.Summary.NavigationEdgeCount,
			routeManifest.Summary.CallChainEdgeCount,
			routeManifest.Summary.SharedRouterHelperCount,
			routeManifest.Summary.TabBarPages,
		)
	}
}

func printASTRenameNotice(options semantic.ASTRenameOptions) {
	lines := semantic.ASTRenameNoticeLines(options)
	if len(lines) == 0 {
		return
	}
	ui.Warning(lines[0])
	for _, line := range lines[1:] {
		ui.Info("   - %s", line)
	}
}
