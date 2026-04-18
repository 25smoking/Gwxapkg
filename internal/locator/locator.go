package locator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/25smoking/Gwxapkg/internal/decrypt"
)

// MiniProgramInfo 存储小程序的基本信息
type MiniProgramInfo struct {
	AppID      string
	AppName    string
	Version    string
	UpdateTime time.Time
	Path       string
	Files      []string
}

// tryReadAppName 尝试读取小程序应用名称
func tryReadAppName(appPath, appID string, files []string) string {
	// 1. 尝试读取原版 appinfo.json（适用于 macOS）
	candidates := []string{"appinfo.json", "appInfo.json", "app-info.json", "info.json"}

	for _, name := range candidates {
		data, err := os.ReadFile(filepath.Join(appPath, name))
		if err != nil {
			continue
		}
		var meta struct {
			Nickname  string `json:"nickname"`
			AppName   string `json:"appName"`
			Name      string `json:"name"`
			Title     string `json:"title"`
		}
		if err := json.Unmarshal(data, &meta); err == nil {
			if meta.Nickname != "" { return meta.Nickname }
			if meta.AppName != "" { return meta.AppName }
			if meta.Name != "" { return meta.Name }
			if meta.Title != "" { return meta.Title }
		}
	}

	// 2. 如果没有找到缓存的 json，尝试从主包直接读取（纯内存，7毫秒）
	// 优先查找 __APP__.wxapkg 主包
	var mainPkg string
	for _, fPath := range files {
		if strings.HasSuffix(fPath, "__APP__.wxapkg") {
			mainPkg = fPath
			break
		}
	}
	
	if mainPkg != "" {
		if name := extractNameFromWxapkg(mainPkg, appID); name != "" {
			return name
		}
	}

	// 如果没有 __APP__.wxapkg，尝试按顺序找任意一个包含 app-config 的包（不中断直到找到）
	for _, fPath := range files {
		if fPath != mainPkg && strings.HasSuffix(fPath, ".wxapkg") {
			if name := extractNameFromWxapkg(fPath, appID); name != "" {
				return name
			}
		}
	}

	return ""
}

// extractNameFromWxapkg 尝试在内存中快速解密并提取包内应用名
func extractNameFromWxapkg(file, appID string) string {
	dec, err := decrypt.DecryptWxapkg(file, appID)
	if err != nil || len(dec) < 14 {
		return ""
	}
	
	r := bytes.NewReader(dec)
	var firstMark byte
	firstMark, _ = r.ReadByte()
	if firstMark != 0xBE {
		return ""
	}
	
	// 跳转到文件数量
	r.Seek(14, 0)
	var buf [4]byte
	if _, err := r.Read(buf[:]); err != nil {
		return ""
	}
	fileCount := uint32(buf[0])<<24 | uint32(buf[1])<<16 | uint32(buf[2])<<8 | uint32(buf[3])
	
	for i := 0; i < int(fileCount); i++ {
		if _, err := r.Read(buf[:]); err != nil { return "" }
		nameLen := uint32(buf[0])<<24 | uint32(buf[1])<<16 | uint32(buf[2])<<8 | uint32(buf[3])
		
		nameBuf := make([]byte, nameLen)
		if _, err := r.Read(nameBuf); err != nil { return "" }
		
		if _, err := r.Read(buf[:]); err != nil { return "" }
		offset := uint32(buf[0])<<24 | uint32(buf[1])<<16 | uint32(buf[2])<<8 | uint32(buf[3])
		
		if _, err := r.Read(buf[:]); err != nil { return "" }
		size := uint32(buf[0])<<24 | uint32(buf[1])<<16 | uint32(buf[2])<<8 | uint32(buf[3])
		
		if string(nameBuf) == "/app-config.json" {
			fileData := make([]byte, size)
			currentPos, _ := r.Seek(0, 1)
			r.Seek(int64(offset), 0)
			r.Read(fileData)
			r.Seek(currentPos, 0) // 恢复偏移量
			
			// 方法1：标准 JSON 解析
			var config map[string]interface{}
			if err := json.Unmarshal(fileData, &config); err == nil {
				if global, ok := config["global"].(map[string]interface{}); ok {
					if window, ok := global["window"].(map[string]interface{}); ok {
						if title, ok := window["navigationBarTitleText"].(string); ok {
							if title != "" { return title }
						}
					}
				}
			}

			// 方法2：直接通过字节搜索，防止 JSON 结构变异或极大
			dataStr := string(fileData)
			
			idx := strings.Index(dataStr, `"navigationBarTitleText":"`)
			if idx != -1 {
				start := idx + 26
				end := strings.IndexByte(dataStr[start:], '"')
				if end != -1 {
					return dataStr[start : start+end]
				}
			}

			idx = strings.Index(dataStr, `"appName":"`)
			if idx != -1 {
				start := idx + 11
				end := strings.IndexByte(dataStr[start:], '"')
				if end != -1 {
					return dataStr[start : start+end]
				}
			}

			return ""
		}
	}
	return ""
}

// Scan 扫描所有可能的微信小程序目录
func Scan() ([]MiniProgramInfo, error) {
	var results []MiniProgramInfo
	var basePaths []string

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %v", err)
	}

	switch runtime.GOOS {
	case "darwin":
		basePaths = collectDarwinBasePaths(homeDir)
	case "windows":
		// Windows 路径
		appData, err := os.UserConfigDir() // 通常是 AppData/Roaming
		if err == nil {
			basePaths = append(basePaths, filepath.Join(appData, "Tencent/xwechat/radium/Applet/packages"))
			
			// 新版微信：用户隔离目录（含用户hash子目录）
			// %APPDATA%\Tencent\xwechat\radium\users\<hash>\applet\packages
			patterns := []string{
				filepath.Join(appData, "Tencent/xwechat/radium/users/*/applet/packages"),
				// %APPDATA%\Tencent\WeChat\radium\Applet\<hash>\packages
				filepath.Join(appData, "Tencent/WeChat/radium/Applet/*/packages"),
			}
			for _, pattern := range patterns {
				matches, err := filepath.Glob(pattern)
				if err != nil {
					continue
				}
				basePaths = append(basePaths, matches...)
			}
		}
		// Documents 路径 (通常是 %USERPROFILE%\Documents)
		// Go 标准库没有直接获取 Documents 的方法，尝试构建
		basePaths = append(basePaths, filepath.Join(homeDir, "Documents/WeChat Files/Applet"))
	}

	seen := make(map[string]struct{})
	for _, basePath := range basePaths {
		if _, ok := seen[basePath]; ok {
			continue
		}
		seen[basePath] = struct{}{}

		if _, err := os.Stat(basePath); err == nil {
			// fmt.Printf("Found WeChat path: %s\n", basePath)
			scanDirectory(basePath, &results)
		}
	}

	// 按时间倒序排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].UpdateTime.After(results[j].UpdateTime)
	})

	return results, nil
}

func collectDarwinBasePaths(homeDir string) []string {
	basePaths := []string{
		// 旧版扫描路径
		filepath.Join(homeDir, "Library/Containers/com.tencent.xinWeChat/Data/Documents/app_data/radium/Applet/packages"),
		// 非沙盒版本
		filepath.Join(homeDir, "Library/Application Support/WeChat/Applet/packages"),
	}

	patterns := []string{
		// 新版微信将小程序缓存放在用户隔离目录下
		filepath.Join(homeDir, "Library/Containers/com.tencent.xinWeChat/Data/Documents/app_data/radium/users/*/applet/packages"),
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		basePaths = append(basePaths, matches...)
	}

	return basePaths
}

func scanDirectory(basePath string, results *[]MiniProgramInfo) {
	// 结构: base_path/{AppID}/{Version}/__APP__.wxapkg

	entries, err := os.ReadDir(basePath)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		appID := entry.Name()
		// 忽略非 AppID 目录
		if !strings.HasPrefix(appID, "wx") {
			continue
		}

		appPath := filepath.Join(basePath, appID)
		verEntries, err := os.ReadDir(appPath)
		if err != nil {
			continue
		}

		for _, verEntry := range verEntries {
			if !verEntry.IsDir() {
				continue
			}

			version := verEntry.Name()
			verPath := filepath.Join(appPath, version)

			var wxapkgFiles []string
			var latestTime time.Time

			err := filepath.WalkDir(verPath, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !d.IsDir() && strings.HasSuffix(d.Name(), ".wxapkg") {
					wxapkgFiles = append(wxapkgFiles, path)

					info, err := d.Info()
					if err == nil {
						if info.ModTime().After(latestTime) {
							latestTime = info.ModTime()
						}
					}
				}
				return nil
			})

			if err != nil {
				continue
			}

			if len(wxapkgFiles) > 0 {
				*results = append(*results, MiniProgramInfo{
					AppID:      appID,
					AppName:    tryReadAppName(appPath, appID, wxapkgFiles),
					Version:    version,
					UpdateTime: latestTime,
					Path:       verPath,
					Files:      wxapkgFiles,
				})
			}
		}
	}
}
