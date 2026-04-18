package config

import (
	"context"
	"os"
	"sync"
)

// FileDeletionManager 用于管理需要删除的文件列表
type FileDeletionManager struct {
	mu       sync.Mutex
	files    map[string]bool
	cancelFn context.CancelFunc
	ctx      context.Context
}

var deleteInstance *FileDeletionManager
var deleteOnce sync.Once

// NewFileDeletionManager 创建或获取一个单例的FileDeletionManager
func NewFileDeletionManager() *FileDeletionManager {
	deleteOnce.Do(func() {
		c := context.Background()
		ctx, cancel := context.WithCancel(c)
		deleteInstance = &FileDeletionManager{
			files:    make(map[string]bool),
			cancelFn: cancel,
			ctx:      ctx,
		}
	})
	return deleteInstance
}

// AddFile 添加文件路径到删除列表
func (f *FileDeletionManager) AddFile(filePath string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.files[filePath] = true
}

// DeleteFiles 删除所有在列表中的文件
func (f *FileDeletionManager) DeleteFiles() {
	f.mu.Lock()
	files := make([]string, 0, len(f.files))
	for file := range f.files {
		files = append(files, file)
	}
	f.mu.Unlock()

	for _, file := range files {
		select {
		case <-f.ctx.Done():
			return
		default:
			if _, err := os.Stat(file); os.IsNotExist(err) {
				continue
			}
			err := os.Remove(file)
			if err == nil {
				f.mu.Lock()
				delete(f.files, file)
				f.mu.Unlock()
			}
		}
	}
}

// Cancel 取消删除操作
func (f *FileDeletionManager) Cancel() {
	f.cancelFn()
}
