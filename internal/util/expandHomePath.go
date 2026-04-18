package util

import (
	"os"
	"path/filepath"
	"strings"
)

// ExpandHomePath 将以 ~ 开头的路径展开为用户主目录。
func ExpandHomePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}

	if path != "~" && !strings.HasPrefix(path, "~/") && !strings.HasPrefix(path, "~\\") {
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	if path == "~" {
		return homeDir, nil
	}

	remainder := strings.TrimPrefix(path, "~")
	remainder = strings.TrimPrefix(remainder, "/")
	remainder = strings.TrimPrefix(remainder, "\\")
	return filepath.Join(homeDir, filepath.FromSlash(remainder)), nil
}
