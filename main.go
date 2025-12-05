package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/25smoking/Gwxapkg/cmd"
	"github.com/25smoking/Gwxapkg/internal/locator"
	"github.com/25smoking/Gwxapkg/internal/pack"
	"github.com/25smoking/Gwxapkg/internal/ui"
)

func main() {
	// 检查是否有子命令
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "all":
			handleAllCommand(os.Args[2:])
			return
		case "scan":
			handleScanCommand()
			return
		case "repack":
			handleRepackCommand(os.Args[2:])
			return
		}
	}

	// 默认命令行模式
	handleDefaultCommand()
}

// handleAllCommand 处理 all 子命令：自动扫描并处理指定 AppID 的所有文件
func handleAllCommand(args []string) {
	allFlags := flag.NewFlagSet("all", flag.ExitOnError)
	appID := allFlags.String("id", "", "微信小程序的AppID")
	outputDir := allFlags.String("out", "", "输出目录路径")
	restoreDir := allFlags.Bool("restore", true, "是否还原工程目录结构")
	pretty := allFlags.Bool("pretty", true, "是否美化输出")
	noClean := allFlags.Bool("noClean", false, "是否保留中间文件")
	save := allFlags.Bool("save", false, "是否保存解密后的文件")
	sensitive := allFlags.Bool("sensitive", true, "是否获取敏感数据")

	allFlags.Parse(args)

	ui.Banner()

	if *appID == "" {
		ui.Error("请指定 AppID: ./Gwxapkg all -id=<AppID>")
		return
	}

	ui.Info("正在扫描 %s 的文件...", *appID)
	fmt.Println()

	programs, err := locator.Scan()
	if err != nil {
		ui.Error("扫描失败: %v", err)
		return
	}

	// 查找匹配的 AppID
	var matched *locator.MiniProgramInfo
	for _, p := range programs {
		if p.AppID == *appID {
			matched = &p
			break
		}
	}

	if matched == nil {
		ui.Error("未找到 AppID: %s", *appID)
		ui.Info("使用 ./Gwxapkg scan 查看所有可用的小程序")
		return
	}

	ui.Success("找到小程序: %s (版本 %s, %d 个文件)", matched.AppID, matched.Version, len(matched.Files))
	ui.PrintDivider()

	// 使用目录路径而非单个文件
	cmd.Execute(*appID, matched.Path, *outputDir, ".wxapkg", *restoreDir, *pretty, *noClean, *save, *sensitive)

	ui.PrintDivider()
	ui.Success("处理完成!")
}

// handleScanCommand 处理 scan 子命令
func handleScanCommand() {
	ui.Banner()
	ui.Info("正在扫描微信小程序目录...")
	fmt.Println()

	programs, err := locator.Scan()
	if err != nil {
		ui.Error("扫描失败: %v", err)
		return
	}

	if len(programs) == 0 {
		ui.Warning("未找到任何微信小程序缓存")
		return
	}

	ui.Success("找到 %d 个小程序", len(programs))
	ui.PrintDivider()
	fmt.Println()

	for i, p := range programs {
		ui.PrintMiniProgram(i+1, p.AppID, p.Version, p.UpdateTime, len(p.Files), p.Path)
	}

	ui.PrintDivider()
	fmt.Println()

	if len(programs) > 0 {
		ui.Info("快速处理示例:")
		fmt.Printf("  ./Gwxapkg all -id=%s\n", programs[0].AppID)
		fmt.Println()
	}
}

// handleRepackCommand 处理 repack 子命令
func handleRepackCommand(args []string) {
	repackFlags := flag.NewFlagSet("repack", flag.ExitOnError)
	inputDir := repackFlags.String("in", "", "输入目录路径")
	outputDir := repackFlags.String("out", "", "输出目录路径")
	watch := repackFlags.Bool("watch", false, "是否监听文件夹")

	repackFlags.Parse(args)

	ui.Banner()

	if *inputDir == "" && len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		*inputDir = args[0]
	}

	if *inputDir == "" {
		ui.Error("请指定输入目录: ./Gwxapkg repack -in=<目录>")
		return
	}

	ui.Info("重新打包模式")
	pack.Repack(*inputDir, *watch, *outputDir)
}

// handleDefaultCommand 处理默认命令行模式
func handleDefaultCommand() {
	appID := flag.String("id", "", "微信小程序的AppID")
	input := flag.String("in", "", "输入文件路径")
	outputDir := flag.String("out", "", "输出目录路径")
	fileExt := flag.String("ext", ".wxapkg", "处理的文件后缀")
	restoreDir := flag.Bool("restore", true, "是否还原工程目录结构")
	pretty := flag.Bool("pretty", true, "是否美化输出")
	noClean := flag.Bool("noClean", false, "是否保留中间文件")
	save := flag.Bool("save", false, "是否保存解密后的文件")
	sensitive := flag.Bool("sensitive", true, "是否获取敏感数据")

	flag.Parse()

	ui.Banner()

	if *appID == "" || *input == "" {
		ui.PrintUsage()
		return
	}

	ui.Info("开始处理小程序: %s", *appID)
	ui.PrintDivider()
	cmd.Execute(*appID, *input, *outputDir, *fileExt, *restoreDir, *pretty, *noClean, *save, *sensitive)
	ui.PrintDivider()
	ui.Success("处理完成!")
}
