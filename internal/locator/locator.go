package locator

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// MiniProgramInfo 存储小程序的基本信息
type MiniProgramInfo struct {
	AppID      string
	Version    string
	UpdateTime time.Time
	Path       string
	Files      []string
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
		// macOS 路径
		basePaths = []string{
			// 新版微信 (沙盒版本)
			filepath.Join(homeDir, "Library/Containers/com.tencent.xinWeChat/Data/Documents/app_data/radium/Applet/packages"),
			// 旧版微信 (非沙盒版本)
			filepath.Join(homeDir, "Library/Application Support/WeChat/Applet/packages"),
		}
	case "windows":
		// Windows 路径
		appData, err := os.UserConfigDir() // 通常是 AppData/Roaming
		if err == nil {
			basePaths = append(basePaths, filepath.Join(appData, "Tencent/xwechat/radium/Applet/packages"))
		}
		// Documents 路径 (通常是 %USERPROFILE%\Documents)
		// Go 标准库没有直接获取 Documents 的方法，尝试构建
		basePaths = append(basePaths, filepath.Join(homeDir, "Documents/WeChat Files/Applet"))
	}

	for _, basePath := range basePaths {
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
					Version:    version,
					UpdateTime: latestTime,
					Path:       verPath,
					Files:      wxapkgFiles,
				})
			}
		}
	}
}
