package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/25smoking/Gwxapkg/cmd"
	internalcmd "github.com/25smoking/Gwxapkg/internal/cmd"
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
		case "scan-only":
			handleScanOnlyCommand(os.Args[2:])
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
// 支持以下方式指定 AppID：
//   - -id=wx111            单个
//   - -id=wx111,wx222      逗号分隔
//   - -id-file=ids.txt     每行一个的文件
//   - --all                处理所有已缓存的小程序
func handleAllCommand(args []string) {
	allFlags := flag.NewFlagSet("all", flag.ExitOnError)
	appID := allFlags.String("id", "", "微信小程序的AppID，支持逗号分隔多个")
	appIDFile := allFlags.String("id-file", "", "AppID 列表文件路径（每行一个）")
	allApps := allFlags.Bool("all", false, "处理所有已缓存的小程序")
	outputDir := allFlags.String("out", "", "输出目录路径")
	restoreDir := allFlags.Bool("restore", true, "是否还原工程目录结构")
	pretty := allFlags.Bool("pretty", true, "是否美化输出")
	noClean := allFlags.Bool("noClean", false, "是否保留中间文件")
	save := allFlags.Bool("save", false, "是否保存解密后的文件")
	sensitive := allFlags.Bool("sensitive", true, "是否获取敏感数据")
	workspace := allFlags.Bool("workspace", false, "是否保留可精确回包的工作区")

	allFlags.Parse(args)

	ui.Banner()

	// 收集 AppID 列表
	var appIDs []string

	if *allApps {
		// --all 模式：扫描所有已缓存小程序
		ui.Info("正在扫描所有已缓存的小程序...")
		programs, err := locator.Scan()
		if err != nil {
			ui.Error("扫描失败: %v", err)
			return
		}
		for _, p := range programs {
			appIDs = append(appIDs, p.AppID)
		}
	} else if *appIDFile != "" {
		// 从文件读取 AppID
		data, err := os.ReadFile(*appIDFile)
		if err != nil {
			ui.Error("读取 AppID 文件失败: %v", err)
			return
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				appIDs = append(appIDs, line)
			}
		}
	} else if *appID != "" {
		// 逗号分隔或单个 AppID
		for _, id := range strings.Split(*appID, ",") {
			id = strings.TrimSpace(id)
			if id != "" {
				appIDs = append(appIDs, id)
			}
		}
	}

	if len(appIDs) == 0 {
		ui.Error("请指定 AppID: ./Gwxapkg all -id=<AppID>")
		ui.Info("或使用 -id-file=ids.txt 指定文件，或 --all 处理全部")
		return
	}

	ui.Info("准备处理 %d 个小程序", len(appIDs))
	fmt.Println()

	// 扫描已缓存的小程序
	programs, err := locator.Scan()
	if err != nil {
		ui.Error("扫描失败: %v", err)
		return
	}

	// 建立 AppID -> MiniProgramInfo 映射
	programMap := make(map[string]*locator.MiniProgramInfo)
	for i := range programs {
		programMap[programs[i].AppID] = &programs[i]
	}

	// 逐个处理
	for i, id := range appIDs {
		if len(appIDs) > 1 {
			ui.PrintDivider()
			ui.Step(i+1, len(appIDs), "处理: %s", id)
		}

		matched, ok := programMap[id]
		if !ok {
			ui.Error("未找到 AppID: %s，跳过", id)
			continue
		}

		displayName := matched.AppID
		if matched.AppName != "" {
			displayName = matched.AppName + " (" + matched.AppID + ")"
		}
		ui.Success("找到小程序: %s （版本 %s, %d 个文件）", displayName, matched.Version, len(matched.Files))

		cmd.Execute(id, matched.Path, *outputDir, ".wxapkg", *restoreDir, *pretty, *noClean, *save, *sensitive, *workspace)
	}

	ui.PrintDivider()
	ui.Success("全部处理完成! (%d 个小程序)", len(appIDs))
}

// handleScanCommand 处理 scan 子命令（交互式选择解包）
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
		ui.PrintMiniProgramWithName(i+1, p.AppID, p.AppName, p.Version, p.UpdateTime, len(p.Files), p.Path)
	}

	ui.PrintDivider()

	// 交互式选择
	choice := ui.Prompt(len(programs))
	if choice == -1 {
		ui.Info("已退出")
		return
	}

	selected := programs[choice-1]
	displayName := selected.AppID
	if selected.AppName != "" {
		displayName = selected.AppName + " (" + selected.AppID + ")"
	}
	ui.Success("已选择: %s", displayName)
	fmt.Println()

	// 直接进入解包流程（复用 all 命令的默认参数）
	cmd.Execute(selected.AppID, selected.Path, "", ".wxapkg", true, true, false, false, true, false)

	ui.PrintDivider()
	ui.Success("处理完成!")
}

// handleScanOnlyCommand 处理 scan-only 子命令
func handleScanOnlyCommand(args []string) {
	f := flag.NewFlagSet("scan-only", flag.ExitOnError)
	dir := f.String("dir", "", "已解包的目录路径")
	appID := f.String("id", "", "AppID（可选，用于报告标题）")
	format := f.String("format", "both", "报告格式: excel / html / both")
	out := f.String("out", "", "报告输出目录（默认与 -dir 相同）")
	f.Parse(args)

	ui.Banner()

	// 支持位置参数
	if *dir == "" && f.NArg() > 0 {
		*dir = f.Arg(0)
	}
	if *dir == "" {
		ui.Error("请指定目录: ./Gwxapkg scan-only -dir=<已解包目录>")
		return
	}

	internalcmd.ScanOnly(*dir, *appID, *format, *out)
}


func handleRepackCommand(args []string) {
	repackFlags := flag.NewFlagSet("repack", flag.ExitOnError)
	inputDir := repackFlags.String("in", "", "输入目录路径")
	outputDir := repackFlags.String("out", "", "输出目录路径")
	watch := repackFlags.Bool("watch", false, "是否监听文件夹")
	appID := repackFlags.String("id", "", "小程序 AppID（用于生成微信可直接打开的加密包）")
	raw := repackFlags.Bool("raw", false, "输出未加密 wxapkg（仅供测试）")

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
	pack.Repack(*inputDir, *watch, *outputDir, *appID, *raw)
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
	workspace := flag.Bool("workspace", false, "是否保留可精确回包的工作区")

	flag.Parse()

	ui.Banner()

	if *appID == "" || *input == "" {
		ui.PrintUsage()
		return
	}

	ui.Info("开始处理小程序: %s", *appID)
	ui.PrintDivider()
	cmd.Execute(*appID, *input, *outputDir, *fileExt, *restoreDir, *pretty, *noClean, *save, *sensitive, *workspace)
	ui.PrintDivider()
	ui.Success("处理完成!")
}
