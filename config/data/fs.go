package data

import (
	"log"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	currentUser   string
	currentUserMu sync.RWMutex
)

// SetCurrentUser 设置当前用户（用于用户数据隔离）
func SetCurrentUser(username string) {
	currentUserMu.Lock()
	defer currentUserMu.Unlock()
	currentUser = username
}

// GetCurrentUser 获取当前用户
func GetCurrentUser() string {
	currentUserMu.RLock()
	defer currentUserMu.RUnlock()
	return currentUser
}

func checkExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func getConfigPath(config string) string {
	rootDir, err := os.Getwd()
	if err != nil {
		return filepath.Join(".", config+".yml")
	}
	// 如果设置了当前用户，始终使用用户目录下的配置文件
	user := GetCurrentUser()
	if user != "" {
		userDir := filepath.Join(rootDir, "users", user)
		// 确保用户目录存在
		if err := os.MkdirAll(userDir, 0755); err != nil {
			log.Printf("[数据隔离] 创建用户目录失败: user=%s, error=%v", user, err)
		}
		return filepath.Join(userDir, config+".yml")
	}
	return filepath.Join(rootDir, config+".yml")
}

func saveFile(filePath string, data []byte) bool {
	// 确保目录存在
	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return false
		}
	}
	err := os.WriteFile(filePath, data, os.ModePerm)
	return err == nil
}

// readFile reads the file and returns (nil, error) on failure. Callers should handle errors.
func readFile(filePath string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return nil, fmt.Errorf("读取配置文件 %s 失败: %w", filePath, err)
	}
	return data, nil
}
