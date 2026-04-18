package pack

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/25smoking/Gwxapkg/internal/config"
)

const (
	manifestDirName  = ".gwxapkg"
	manifestFileName = "manifest.json"
)

// PackageManifest 保存原始多包结构，供后续精确回包使用
type PackageManifest struct {
	Version     int               `json:"version"`
	AppID       string            `json:"app_id"`
	GeneratedAt string            `json:"generated_at"`
	Packages    []ManifestPackage `json:"packages"`
}

// ManifestPackage 描述单个原始 wxapkg 包
type ManifestPackage struct {
	Name       string   `json:"name"`
	SourceRoot string   `json:"source_root,omitempty"`
	Files      []string `json:"files"`
}

func WritePackageManifest(outputDir string, appID string, manager *config.WxapkgManager) error {
	if manager == nil {
		return fmt.Errorf("包管理器为空")
	}

	manifest := &PackageManifest{
		Version:     1,
		AppID:       appID,
		GeneratedAt: time.Now().Format(time.RFC3339),
	}

	for _, pkg := range manager.SnapshotPackages() {
		if pkg == nil || pkg.PackageName == "" || len(pkg.RawFiles) == 0 {
			continue
		}

		files := normalizeManifestFiles(pkg.RawFiles)
		if len(files) == 0 {
			continue
		}

		manifest.Packages = append(manifest.Packages, ManifestPackage{
			Name:       pkg.PackageName,
			SourceRoot: pkg.RawRoot,
			Files:      files,
		})
	}

	if len(manifest.Packages) == 0 {
		return nil
	}

	sort.Slice(manifest.Packages, func(i, j int) bool {
		return manifestPackageOrder(manifest.Packages[i].Name) < manifestPackageOrder(manifest.Packages[j].Name)
	})

	manifestDir := filepath.Join(outputDir, manifestDirName)
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		return fmt.Errorf("创建 manifest 目录失败: %w", err)
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 manifest 失败: %w", err)
	}

	if err := os.WriteFile(filepath.Join(manifestDir, manifestFileName), data, 0644); err != nil {
		return fmt.Errorf("写入 manifest 失败: %w", err)
	}

	return nil
}

func LoadPackageManifest(inputDir string) (*PackageManifest, error) {
	manifestPath := filepath.Join(inputDir, manifestDirName, manifestFileName)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var manifest PackageManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("解析 manifest 失败: %w", err)
	}

	return &manifest, nil
}

func normalizeManifestFiles(files []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(files))

	for _, file := range files {
		normalized := strings.TrimPrefix(filepath.ToSlash(file), "/")
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}

	return result
}

func manifestPackageOrder(name string) string {
	if name == "__APP__.wxapkg" {
		return "0-" + name
	}
	return "1-" + name
}
