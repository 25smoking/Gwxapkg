package ui

import (
	"fmt"
	"os"
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
	dim.Println("              Wxapkg Decompiler Tool v2.5.1")
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
	fmt.Printf("  %s %s\n", cyan.Sprintf("%2d.", index), green.Sprint(appID))
	dim.Printf("     版本: %s │ 文件: %d │ 更新: %s\n", version, fileCount, updateTime.Format("2006-01-02 15:04"))
	dim.Printf("     路径: %s\n\n", path)
}

// PrintDivider 打印分隔线
func PrintDivider() {
	dim.Println("─────────────────────────────────────────────────────")
}

// PrintUsage 打印使用帮助
func PrintUsage() {
	cyan.Println("命令:")
	fmt.Println()
	white.Println("  scan                     扫描本地小程序")
	white.Println("  all -id=<AppID>          自动查找并处理指定小程序")
	white.Println("  repack -in=<目录>        重新打包")
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
	fmt.Println()
}
