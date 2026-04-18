package config

import (
	"sync"

	"github.com/25smoking/Gwxapkg/internal/enum"
)

// Parser 接口
type Parser interface {
	Parse(option WxapkgInfo) error
}

// WxapkgOption 微信小程序解包选项
type WxapkgOption struct {
	ViewSource      string
	AppConfigSource string
	ServiceSource   string
	SetAppConfig    bool
}

// WxapkgInfo 保存包的信息
type WxapkgInfo struct {
	WxAppId     string
	WxapkgType  enum.WxapkgType
	PackageName string
	SourcePath  string
	RawFiles    []string
	RawRoot     string
	IsExtracted bool
	Option      *WxapkgOption
	Parsers     []Parser // 添加解析器列表
}

// WxapkgManager 管理多个微信小程序包
type WxapkgManager struct {
	mu       sync.RWMutex
	Packages map[string]*WxapkgInfo
}

var managerInstance *WxapkgManager
var wxapkgOnce sync.Once

// GetWxapkgManager 获取单例的 WxapkgManager 实例
func GetWxapkgManager() *WxapkgManager {
	wxapkgOnce.Do(func() {
		managerInstance = &WxapkgManager{
			Packages: make(map[string]*WxapkgInfo),
		}
	})
	return managerInstance
}

// AddPackage 添加包信息
func (manager *WxapkgManager) AddPackage(id string, info *WxapkgInfo) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	manager.Packages[id] = info
}

// GetPackage 获取包信息
func (manager *WxapkgManager) GetPackage(id string) (*WxapkgInfo, bool) {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	info, exists := manager.Packages[id]
	return info, exists
}

// SnapshotPackages 返回当前包信息快照，避免并发读写 map
func (manager *WxapkgManager) SnapshotPackages() []*WxapkgInfo {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	result := make([]*WxapkgInfo, 0, len(manager.Packages))
	for _, info := range manager.Packages {
		result = append(result, info)
	}
	return result
}
