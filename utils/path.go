package utils

import (
	"path/filepath"
	"runtime"
	"strings"
)

// 通用路径处理（兼容不同操作系统）
func GetProjectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..")
}

// 安全路径拼接
func SafeJoin(basePath, subPath string) string {
	p := filepath.Join(basePath, subPath)
	// 防止路径穿越攻击
	if !strings.HasPrefix(p, filepath.Clean(basePath)) {
		return ""
	}
	return p
}
