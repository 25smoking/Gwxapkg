package cmd

import (
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"

	. "github.com/25smoking/Gwxapkg/internal/cmd"
	. "github.com/25smoking/Gwxapkg/internal/config"
	"github.com/25smoking/Gwxapkg/internal/restore"
	"github.com/25smoking/Gwxapkg/internal/ui"
)

func Execute(appID, input, outputDir, fileExt string, restoreDir bool, pretty bool, noClean bool, save bool, sensitive bool) {
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

	inputFiles := ParseInput(input, fileExt)

	if len(inputFiles) == 0 {
		ui.Warning("未找到任何文件")
		return
	}

	// 确定输出目录
	if outputDir == "" {
		outputDir = DetermineOutputDir(input, appID)
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
			err := ProcessFile(file, outputDir, appID, save)
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

	// 还原工程目录结构
	ui.Step(2, 2, "还原工程结构...")
	restore.ProjectStructure(outputDir, restoreDir)

	// 输出结果目录
	fmt.Println()
	ui.Success("输出目录: %s", filepath.Clean(outputDir))
}
