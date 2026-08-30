package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"
)

// UserRecord 用户记录
type UserRecord struct {
	Username          string `yaml:"username"`
	PasswordHash      string `yaml:"password_hash"`
	IsAdmin           bool   `yaml:"is_admin"`
	MustChangePassword bool  `yaml:"must_change_password"`
}

// UsersData 用户数据
type UsersData struct {
	Users []UserRecord `yaml:"users"`
}

var (
	usersCache *UsersData
	usersMu    sync.RWMutex
)

// getUsersFilePath 获取用户数据文件路径
func getUsersFilePath() string {
	workDir, err := os.Getwd()
	if err != nil {
		return "users.yml"
	}
	return filepath.Join(workDir, "users.yml")
}

// loadUsers 加载用户数据
func loadUsers() (*UsersData, error) {
	usersMu.RLock()
	if usersCache != nil {
		defer usersMu.RUnlock()
		return usersCache, nil
	}
	usersMu.RUnlock()

	usersMu.Lock()
	defer usersMu.Unlock()

	if usersCache != nil {
		return usersCache, nil
	}

	data := &UsersData{Users: []UserRecord{}}
	filePath := getUsersFilePath()
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// 文件不存在，创建空文件
		if err := saveUsers(data); err != nil {
			return nil, err
		}
		usersCache = data
		return data, nil
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(content, data); err != nil {
		return nil, err
	}

	usersCache = data
	return data, nil
}

// saveUsers 保存用户数据
func saveUsers(data *UsersData) error {
	content, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	filePath := getUsersFilePath()
	return os.WriteFile(filePath, content, 0644)
}

// HashPassword 密码哈希
func HashPassword(password string) string {
	hash := sha256.Sum256([]byte(password + "flare_salt_2024"))
	return hex.EncodeToString(hash[:])
}

// UserExists 检查用户名是否已存在
func UserExists(username string) (bool, error) {
	users, err := loadUsers()
	if err != nil {
		return false, err
	}
	username = strings.TrimSpace(username)
	for _, u := range users.Users {
		if strings.EqualFold(u.Username, username) {
			return true, nil
		}
	}
	return false, nil
}

// RegisterUser 注册新用户
func RegisterUser(username, password string) error {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return fmt.Errorf("用户名或密码不能为空")
	}

	exists, err := UserExists(username)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("用户名已存在")
	}

	usersMu.Lock()
	defer usersMu.Unlock()

	if usersCache == nil {
		usersCache = &UsersData{Users: []UserRecord{}}
	}

	usersCache.Users = append(usersCache.Users, UserRecord{
		Username:     username,
		PasswordHash: HashPassword(password),
	})

	return saveUsers(usersCache)
}

// VerifyUser 验证用户密码
func VerifyUser(username, password string) bool {
	username = strings.TrimSpace(username)
	users, err := loadUsers()
	if err != nil {
		return false
	}
	hash := HashPassword(password)
	for _, u := range users.Users {
		if strings.EqualFold(u.Username, username) && u.PasswordHash == hash {
			return true
		}
	}
	return false
}

// GetUserDir 获取用户数据目录
func GetUserDir(username string) string {
	workDir, err := os.Getwd()
	if err != nil {
		return filepath.Join("users", username)
	}
	return filepath.Join(workDir, "users", username)
}

// InitUserDir 初始化用户数据目录
func InitUserDir(username string) error {
	userDir := GetUserDir(username)
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return err
	}
	// 复制默认配置文件到用户目录
	workDir, err := os.Getwd()
	if err != nil {
		return nil
	}
	for _, name := range []string{"apps.yml", "bookmarks.yml", "config.yml"} {
		src := filepath.Join(workDir, name)
		dst := filepath.Join(userDir, name)
		if _, err := os.Stat(src); err == nil {
			if _, err := os.Stat(dst); os.IsNotExist(err) {
				data, err := os.ReadFile(src)
				if err == nil {
					os.WriteFile(dst, data, 0644)
				}
			}
		}
	}
	return nil
}
