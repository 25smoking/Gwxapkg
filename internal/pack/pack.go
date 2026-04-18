package pack

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fsnotify/fsnotify"

	"github.com/25smoking/Gwxapkg/internal/decrypt"
	"github.com/25smoking/Gwxapkg/internal/util"
)

func Repack(path string, watch bool, outputPath string, appID string, raw bool) {
	// 过滤空白字符
	path = strings.TrimSpace(path)
	outputPath = strings.TrimSpace(outputPath)
	appID = strings.TrimSpace(appID)

	expandedOutputPath, err := util.ExpandHomePath(outputPath)
	if err != nil {
		log.Printf("警告: 展开输出目录失败，继续使用原路径: %v\n", err)
	} else {
		outputPath = expandedOutputPath
	}

	// 如果是目录，则打包目录
	if fileInfo, err := os.Stat(path); err != nil || !fileInfo.IsDir() {
		log.Printf("错误: %s 不是一个有效的目录\n", path)
		return
	}

	// 优先按 manifest 精确恢复原始多包结构
	if handled, err := repackWithManifest(path, outputPath, appID, raw); err != nil {
		log.Printf("错误: %v\n", err)
		return
	} else if handled {
		if watch {
			watchDir(path, outputPath, appID, raw)
		}
		return
	}

	// 打包目录
	outputFile, err := packWxapkg(path, outputPath, appID, raw)
	if err != nil {
		log.Printf("错误: %v\n", err)
		return
	}
	log.Printf("打包完成: %s\n", outputFile)

	if watch {
		watchDir(path, outputPath, appID, raw)
	}

	return
}

type WxapkgFile struct {
	NameLen uint32
	Name    string
	Offset  uint32
	Size    uint32
	Source  string
}

// 打包文件到 wxapkg 格式
func packWxapkg(inputDir string, outputPath string, appID string, raw bool) (string, error) {
	outputFile, err := resolveOutputFile(inputDir, outputPath)
	if err != nil {
		return "", err
	}

	files, err := collectAllFiles(inputDir)
	if err != nil {
		return "", err
	}

	return packFiles(files, outputFile, appID, raw)
}

func collectAllFiles(inputDir string) ([]WxapkgFile, error) {
	relPaths := make([]string, 0)
	err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(inputDir, path)
		if err != nil {
			return err
		}
		normalizedRelPath := filepath.ToSlash(relPath)

		// 排除隐藏工作区，避免把 manifest 和原始运行时文件再次打入包内
		if normalizedRelPath == ".gwxapkg" || strings.HasPrefix(normalizedRelPath, ".gwxapkg/") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 排除目录
		if info.IsDir() {
			return nil
		}

		// 排除 .wxapkg 文件
		if filepath.Ext(path) == ".wxapkg" {
			return nil
		}

		relPaths = append(relPaths, normalizedRelPath)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("计算文件列表失败: %w", err)
	}

	sort.Strings(relPaths)
	return buildFilesFromRelativePaths(inputDir, relPaths)
}

func buildFilesFromRelativePaths(inputDir string, relPaths []string) ([]WxapkgFile, error) {
	files := make([]WxapkgFile, 0, len(relPaths))
	var totalSize uint32
	var missing []string

	for _, relPath := range relPaths {
		normalized := filepath.ToSlash(strings.TrimPrefix(relPath, "/"))
		fullPath := filepath.Join(inputDir, filepath.FromSlash(normalized))
		info, err := os.Stat(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				missing = append(missing, normalized)
				continue
			}
			return nil, fmt.Errorf("读取文件信息失败 %s: %w", normalized, err)
		}
		if info.IsDir() {
			continue
		}

		name := "/" + normalized
		files = append(files, WxapkgFile{
			NameLen: uint32(len(name)),
			Name:    name,
			Offset:  totalSize,
			Size:    uint32(info.Size()),
			Source:  fullPath,
		})
		totalSize += uint32(info.Size())
	}

	if len(missing) > 0 {
		preview := missing
		if len(preview) > 5 {
			preview = preview[:5]
		}
		return nil, fmt.Errorf("缺少 manifest 中记录的文件，共 %d 个，例如: %s", len(missing), strings.Join(preview, ", "))
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("未找到任何可打包文件")
	}

	return files, nil
}

func packFiles(files []WxapkgFile, outputFile string, appID string, raw bool) (string, error) {
	var totalSize uint32
	for _, file := range files {
		totalSize += file.Size
	}

	// 创建输出文件
	outFile, err := os.Create(outputFile)
	if err != nil {
		return "", fmt.Errorf("创建输出文件失败: %w", err)
	}
	closed := false
	defer func(outFile *os.File) {
		if closed {
			return
		}
		err := outFile.Close()
		if err != nil {
			log.Printf("关闭输出文件失败: %v\n", err)
		}
	}(outFile)

	// 写入文件头
	if err := binary.Write(outFile, binary.BigEndian, byte(0xBE)); err != nil {
		return "", fmt.Errorf("写入文件头标记失败: %w", err)
	}

	info1 := uint32(0) // 示例值
	if err := binary.Write(outFile, binary.BigEndian, info1); err != nil {
		return "", fmt.Errorf("写入 info1 失败: %w", err)
	}

	// 计算索引段长度，包含每个文件的元数据长度和文件名长度
	var indexInfoLength uint32
	for _, file := range files {
		indexInfoLength += 4 + uint32(len(file.Name)) + 4 + 4 // NameLen + Name + Offset + Size
	}

	if err := binary.Write(outFile, binary.BigEndian, indexInfoLength); err != nil {
		return "", fmt.Errorf("写入索引段长度失败: %w", err)
	}

	bodyInfoLength := totalSize
	if err := binary.Write(outFile, binary.BigEndian, bodyInfoLength); err != nil {
		return "", fmt.Errorf("写入数据段长度失败: %w", err)
	}

	if err := binary.Write(outFile, binary.BigEndian, byte(0xED)); err != nil {
		return "", fmt.Errorf("写入文件尾标记失败: %w", err)
	}

	// 写入文件数量
	fileCount := uint32(len(files))
	if err := binary.Write(outFile, binary.BigEndian, fileCount); err != nil {
		return "", fmt.Errorf("写入文件数量失败: %w", err)
	}

	// 写入索引段
	for _, file := range files {
		if err := binary.Write(outFile, binary.BigEndian, file.NameLen); err != nil {
			return "", fmt.Errorf("写入文件名长度失败: %w", err)
		}
		if _, err := outFile.Write([]byte(file.Name)); err != nil {
			return "", fmt.Errorf("写入文件名失败: %w", err)
		}
		// 加上 18 字节文件头长度和索引段长度
		if err := binary.Write(outFile, binary.BigEndian, file.Offset+indexInfoLength+18); err != nil {
			return "", fmt.Errorf("写入文件偏移量失败: %w", err)
		}
		if err := binary.Write(outFile, binary.BigEndian, file.Size); err != nil {
			return "", fmt.Errorf("写入文件大小失败: %w", err)
		}
	}

	// 写入数据段
	for _, file := range files {
		func() {
			f, err := os.Open(file.Source)
			if err != nil {
				log.Printf("打开文件失败: %v\n", err)
				return
			}
			defer func(f *os.File) {
				err := f.Close()
				if err != nil {
					log.Printf("关闭文件失败: %v\n", err)
				}
			}(f)

			if _, err = io.Copy(outFile, f); err != nil {
				log.Printf("写入文件内容失败: %v\n", err)
			}
		}()
	}

	if err := outFile.Close(); err != nil {
		return "", fmt.Errorf("关闭输出文件失败: %w", err)
	}
	closed = true

	if raw {
		log.Println("警告: 当前输出为未加密 wxapkg，仅适合工具链测试，微信客户端通常无法直接打开")
		return outputFile, nil
	}

	if appID == "" {
		log.Println("警告: 未提供 AppID，已输出未加密 wxapkg；如需在微信客户端中使用，请追加 -id=<AppID>")
		return outputFile, nil
	}

	rawData, err := os.ReadFile(outputFile)
	if err != nil {
		return "", fmt.Errorf("读取未加密包失败: %w", err)
	}

	encryptedData, err := decrypt.EncryptWxapkg(rawData, appID)
	if err != nil {
		return "", fmt.Errorf("加密 wxapkg 失败: %w", err)
	}

	if err := os.WriteFile(outputFile, encryptedData, 0644); err != nil {
		return "", fmt.Errorf("写入加密包失败: %w", err)
	}

	return outputFile, nil
}

func repackWithManifest(inputDir string, outputPath string, appID string, raw bool) (bool, error) {
	manifest, err := LoadPackageManifest(inputDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	if len(manifest.Packages) == 0 {
		return false, nil
	}

	if appID == "" {
		appID = strings.TrimSpace(manifest.AppID)
	}

	outputDir, err := resolveMultiPackageOutputDir(inputDir, outputPath)
	if err != nil {
		return false, err
	}

	for _, pkg := range manifest.Packages {
		baseDir := inputDir
		if pkg.SourceRoot != "" {
			if filepath.IsAbs(pkg.SourceRoot) {
				baseDir = filepath.Clean(pkg.SourceRoot)
			} else {
				baseDir = filepath.Join(inputDir, filepath.FromSlash(pkg.SourceRoot))
			}
		}

		files, err := buildFilesFromRelativePaths(baseDir, pkg.Files)
		if err != nil {
			return false, fmt.Errorf("构建包 %s 失败: %w", pkg.Name, err)
		}

		outputFile := filepath.Join(outputDir, pkg.Name)
		if _, err := packFiles(files, outputFile, appID, raw); err != nil {
			return false, fmt.Errorf("写出包 %s 失败: %w", pkg.Name, err)
		}
	}

	log.Printf("已按 manifest 生成 %d 个包: %s\n", len(manifest.Packages), outputDir)
	return true, nil
}

func resolveOutputFile(inputDir string, outputPath string) (string, error) {
	defaultName := filepath.Base(filepath.Clean(inputDir)) + ".wxapkg"

	if outputPath == "" {
		return filepath.Join(filepath.Dir(filepath.Clean(inputDir)), defaultName), nil
	}

	outputInfo, err := os.Stat(outputPath)
	if err != nil {
		if os.IsNotExist(err) {
			if filepath.Ext(outputPath) == "" {
				if err := os.MkdirAll(outputPath, 0755); err != nil {
					return "", fmt.Errorf("无法创建输出目录: %w", err)
				}
				return filepath.Join(outputPath, defaultName), nil
			}

			parent := filepath.Dir(outputPath)
			if parent != "." && parent != "" {
				if err := os.MkdirAll(parent, 0755); err != nil {
					return "", fmt.Errorf("无法创建输出目录: %w", err)
				}
			}
			return outputPath, nil
		}

		return "", fmt.Errorf("无法访问输出路径: %w", err)
	}

	if outputInfo.IsDir() {
		return filepath.Join(outputPath, defaultName), nil
	}

	return outputPath, nil
}

func resolveMultiPackageOutputDir(inputDir string, outputPath string) (string, error) {
	defaultDir := filepath.Join(filepath.Dir(filepath.Clean(inputDir)), filepath.Base(filepath.Clean(inputDir))+"_repacked")

	if outputPath == "" {
		if err := os.MkdirAll(defaultDir, 0755); err != nil {
			return "", fmt.Errorf("无法创建输出目录: %w", err)
		}
		return defaultDir, nil
	}

	outputInfo, err := os.Stat(outputPath)
	if err != nil {
		if os.IsNotExist(err) {
			if filepath.Ext(outputPath) != "" {
				return "", fmt.Errorf("多包回包模式下 -out 必须是目录路径")
			}
			if err := os.MkdirAll(outputPath, 0755); err != nil {
				return "", fmt.Errorf("无法创建输出目录: %w", err)
			}
			return outputPath, nil
		}
		return "", fmt.Errorf("无法访问输出目录: %w", err)
	}

	if !outputInfo.IsDir() {
		return "", fmt.Errorf("多包回包模式下 -out 必须是目录路径")
	}

	return outputPath, nil
}

func watchDir(inputDir string, outputPath string, appID string, raw bool) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println("ERROR: ", err)
		return
	}
	defer func(watcher *fsnotify.Watcher) {
		err := watcher.Close()
		if err != nil {
			log.Println("ERROR: ", err)
		}
	}(watcher)

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Remove == fsnotify.Remove {
					log.Println("检测到文件变化: ", event.Name)
					if handled, err := repackWithManifest(inputDir, outputPath, appID, raw); err != nil {
						log.Println("打包失败: ", err)
					} else if handled {
						log.Println("按 manifest 回包成功")
					} else if _, err := packWxapkg(inputDir, outputPath, appID, raw); err != nil {
						log.Println("打包失败: ", err)
					} else {
						log.Println("打包成功")
					}
				}
			case err := <-watcher.Errors:
				log.Println("ERROR: ", err)
			}
		}
	}()

	err = watcher.Add(inputDir)
	if err != nil {
		log.Println("ERROR: ", err)
	}
	<-done
}
