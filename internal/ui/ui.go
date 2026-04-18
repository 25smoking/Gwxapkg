package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
)

var (
	// 颜色定义
	cyan    = color.New(color.FgCyan, color.Bold)
	green   = color.New(color.FgGreen, color.Bold)
	yellow  = color.New(color.FgYellow, color.Bold)
	red     = color.New(color.FgRed, color.Bold)
	magenta = color.New(color.FgMagenta, color.Bold)
	white   = color.New(color.FgWhite)
	dim     = color.New(color.FgHiBlack)
)

// Banner 打印程序横幅
func Banner() {
	cyan.Println(`
  ██████╗ ██╗    ██╗██╗  ██╗ █████╗ ██████╗ ██╗  ██╗ ██████╗ 
 ██╔════╝ ██║    ██║╚██╗██╔╝██╔══██╗██╔══██╗██║ ██╔╝██╔════╝ 
 ██║  ███╗██║ █╗ ██║ ╚███╔╝ ███████║██████╔╝█████╔╝ ██║  ███╗
 ██║   ██║██║███╗██║ ██╔██╗ ██╔══██║██╔═══╝ ██╔═██╗ ██║   ██║
 ╚██████╔╝╚███╔███╔╝██╔╝ ██╗██║  ██║██║     ██║  ██╗╚██████╔╝
  ╚═════╝  ╚══╝╚══╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝     ╚═╝  ╚═╝ ╚═════╝`)
	dim.Println("              Wxapkg Decompiler Tool v2.7.0")
	fmt.Println()
}

// Success 打印成功消息
func Success(format string, a ...interface{}) {
	green.Print("✓ ")
	white.Printf(format+"\n", a...)
}

// Info 打印信息消息
func Info(format string, a ...interface{}) {
	cyan.Print("ℹ ")
	white.Printf(format+"\n", a...)
}

// Warning 打印警告消息
func Warning(format string, a ...interface{}) {
	yellow.Print("⚠ ")
	white.Printf(format+"\n", a...)
}

// Error 打印错误消息
func Error(format string, a ...interface{}) {
	red.Print("✗ ")
	white.Printf(format+"\n", a...)
}

// Step 打印步骤
func Step(step int, total int, format string, a ...interface{}) {
	magenta.Printf("[%d/%d] ", step, total)
	white.Printf(format+"\n", a...)
}

// NewProgressBar 创建新的进度条
func NewProgressBar(max int, description string) *progressbar.ProgressBar {
	return progressbar.NewOptions(max,
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(false),
		progressbar.OptionSetWidth(40),
		progressbar.OptionSetDescription(fmt.Sprintf("[cyan]%s[reset]", description)),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]█[reset]",
			SaucerHead:    "[green]▓[reset]",
			SaucerPadding: "░",
			BarStart:      "│",
			BarEnd:        "│",
		}),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			fmt.Println()
		}),
	)
}

// NewSpinner 创建简单的加载动画
func NewSpinner(description string) *progressbar.ProgressBar {
	return progressbar.NewOptions(-1,
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetDescription(fmt.Sprintf("[cyan]%s[reset]", description)),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionSetRenderBlankState(true),
	)
}

// PrintMiniProgram 美化打印小程序信息
func PrintMiniProgram(index int, appID, version string, updateTime time.Time, fileCount int, path string) {
	PrintMiniProgramWithName(index, appID, "", version, updateTime, fileCount, path)
}

// PrintMiniProgramWithName 美化打印小程序信息（含应用名）
func PrintMiniProgramWithName(index int, appID, appName, version string, updateTime time.Time, fileCount int, path string) {
	nameStr := ""
	if appName != "" {
		nameStr = "  " + magenta.Sprint(appName)
	}
	fmt.Printf("  %s %s%s\n", cyan.Sprintf("%2d.", index), green.Sprint(appID), nameStr)
	dim.Printf("     版本: %s │ 文件: %d │ 更新: %s\n", version, fileCount, updateTime.Format("2006-01-02 15:04"))
	dim.Printf("     路径: %s\n\n", path)
}

// Prompt 显示提示并读取用户输入的编号，返回选择的索引 (1-based)。
// 输入 q 返回 -1，输入无效时重新提示。
func Prompt(maxIndex int) int {
	reader := bufio.NewReader(os.Stdin)
	for {
		yellow.Print("\n请选择要处理的小程序编号（输入 q 退出）: ")
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "q" || line == "Q" {
			return -1
		}
		n, err := strconv.Atoi(line)
		if err != nil || n < 1 || n > maxIndex {
			red.Printf("无效输入，请输入 1-%d 之间的数字或 q 退出\n", maxIndex)
			continue
		}
		return n
	}
}

// PrintDivider 打印分隔线
func PrintDivider() {
	dim.Println("─────────────────────────────────────────────────────")
}

// PrintUsage 打印使用帮助
func PrintUsage() {
	cyan.Println("命令:")
	fmt.Println()
	white.Println("  scan                          扫描本地小程序（交互式选择解包）")
	white.Println("  all -id=<AppID>               自动查找并处理指定小程序")
	white.Println("  all -id=wx1,wx2,wx3           批量处理（逗号分隔）")
	white.Println("  all -id-file=ids.txt          批量处理（文件，每行一个 AppID）")
	white.Println("  all --all                     处理所有已缓存的小程序")
	white.Println("  scan-only -dir=<目录>          对已解包目录独立扫描并生成报告")
	white.Println("  repack -in=<目录> -id=<AppID>  重新打包为客户端可用 wxapkg")
	fmt.Println()
	cyan.Println("直接使用:")
	dim.Println("  ./Gwxapkg -id=<AppID> -in=<文件路径>")
	fmt.Println()
	cyan.Println("可选参数:")
	dim.Println("  -out         输出目录")
	dim.Println("  -restore     还原目录结构 (默认: true)")
	dim.Println("  -pretty      美化代码输出 (默认: true)")
	dim.Println("  -noClean     保留中间文件 (默认: false)")
	dim.Println("  -save        保存解密文件 (默认: false)")
	dim.Println("  -sensitive   获取敏感数据 (默认: true)")
	dim.Println("  -workspace   保留可精确回包的隐藏工作区 (默认: false)")
	dim.Println("  repack -id   生成加密包，适用于回写微信客户端")
	dim.Println("  repack -raw  生成未加密包，仅供测试")
	dim.Println("  scan-only -format  报告格式: excel / html / both (默认: both)")
	fmt.Println()
}
