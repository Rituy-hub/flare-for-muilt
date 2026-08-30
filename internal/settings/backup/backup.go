package backup

import (
	"archive/zip"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/soulteary/flare/config/data"
	"github.com/soulteary/flare/config/define"
	"github.com/soulteary/flare/internal/auth"
	"github.com/soulteary/flare/internal/pool"
	version "github.com/soulteary/version-kit"
)

// 需要备份恢复的配置文件名称
var backupFiles = []string{"apps", "bookmarks", "config"}

func RegisterRouting(e *echo.Echo) {
	e.GET(define.SettingPages.Backup.Path, pageBackup)
	e.GET(define.SettingPages.Backup.Path+"/export", exportBackup)
	e.POST(define.SettingPages.Backup.Path+"/import", importBackup)
}

func pageBackup(c *echo.Context) error {
	options, err := data.GetAllSettingsOptions()
	if err != nil {
		return c.String(http.StatusInternalServerError, "config error")
	}
	locale := options.Locale
	if locale == "" {
		locale = "zh"
	}
	isLogined := false
	if !define.AppFlags.DisableLoginMode {
		isLogined = auth.CheckUserIsLogin(c)
	} else {
		isLogined = true
	}
	m := pool.GetTemplateMap()
	defer pool.PutTemplateMap(m)
	m["Locale"] = locale
	m["DebugMode"] = define.AppFlags.DebugMode
	m["DisableLoginMode"] = define.AppFlags.DisableLoginMode
	m["UserIsLogin"] = isLogined
	m["UserName"] = auth.GetUserName(c)
	m["LoginDate"] = auth.GetUserLoginDate(c)
	m["PageInlineStyle"] = define.GetPageInlineStyle()
	m["PageAppearance"] = define.GetAppBodyStyle()
	m["SettingsURI"] = define.RegularPages.Settings.Path
	m["LoginURI"] = define.MiscPages.Login.Path
	m["LogoutURI"] = define.MiscPages.Logout.Path
	m["PageName"] = "Backup"
	m["SettingPages"] = define.SettingPages
	m["OptionTitle"] = options.Title
	m["Version"] = version.Version
	m["BuildDate"] = version.BuildDate
	m["COMMIT"] = version.Commit
	m["OptionFooter"] = template.HTML(options.Footer)
	m["BackupExportURI"] = define.SettingPages.Backup.Path + "/export"
	m["BackupImportURI"] = define.SettingPages.Backup.Path + "/import"
	return c.Render(http.StatusOK, "settings.html", m)
}

// exportBackup 导出三个配置文件为zip下载
func exportBackup(c *echo.Context) error {
	// 检查登录状态
	if !define.AppFlags.DisableLoginMode && !auth.CheckUserIsLogin(c) {
		return c.String(http.StatusUnauthorized, "请先登录")
	}

	workDir, err := os.Getwd()
	if err != nil {
		return c.String(http.StatusInternalServerError, "获取工作目录失败")
	}

	// 创建临时zip文件
	tmpFile, err := os.CreateTemp("", "flare-backup-*.zip")
	if err != nil {
		return c.String(http.StatusInternalServerError, "创建临时文件失败")
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	zipWriter := zip.NewWriter(tmpFile)

	for _, name := range backupFiles {
		filePath := filepath.Join(workDir, name+".yml")
		fileData, readErr := os.ReadFile(filePath)
		if readErr != nil {
			// 文件不存在则跳过
			continue
		}
		f, createErr := zipWriter.Create(name + ".yml")
		if createErr != nil {
			continue
		}
		if _, writeErr := f.Write(fileData); writeErr != nil {
			continue
		}
	}

	if closeErr := zipWriter.Close(); closeErr != nil {
		return c.String(http.StatusInternalServerError, "压缩文件失败")
	}
	tmpFile.Close()

	// 读取zip文件内容
	zipData, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return c.String(http.StatusInternalServerError, "读取备份文件失败")
	}

	// 设置下载响应头
	c.Response().Header().Set("Content-Type", "application/zip")
	c.Response().Header().Set("Content-Disposition", "attachment; filename=flare-backup.zip")
	c.Response().Header().Set("Content-Length", fmt.Sprintf("%d", len(zipData)))
	return c.Blob(http.StatusOK, "application/zip", zipData)
}

// importBackup 从上传的zip文件恢复三个配置文件
func importBackup(c *echo.Context) error {
	// 检查登录状态
	if !define.AppFlags.DisableLoginMode && !auth.CheckUserIsLogin(c) {
		return c.String(http.StatusUnauthorized, "请先登录")
	}

	// 获取上传的文件
	file, err := c.FormFile("backup_file")
	if err != nil {
		return c.String(http.StatusBadRequest, "请选择要恢复的备份文件")
	}

	src, err := file.Open()
	if err != nil {
		return c.String(http.StatusInternalServerError, "打开上传文件失败")
	}
	defer src.Close()

	// 创建临时文件保存上传的zip
	tmpFile, err := os.CreateTemp("", "flare-import-*.zip")
	if err != nil {
		return c.String(http.StatusInternalServerError, "创建临时文件失败")
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err = io.Copy(tmpFile, src); err != nil {
		return c.String(http.StatusInternalServerError, "保存上传文件失败")
	}
	tmpFile.Close()

	// 打开zip文件
	zipReader, err := zip.OpenReader(tmpFile.Name())
	if err != nil {
		return c.String(http.StatusBadRequest, "备份文件格式错误，请上传有效的zip文件")
	}
	defer zipReader.Close()

	workDir, err := os.Getwd()
	if err != nil {
		return c.String(http.StatusInternalServerError, "获取工作目录失败")
	}

	restoredCount := 0
	for _, f := range zipReader.File {
		// 只处理yml文件
		if !strings.HasSuffix(f.Name, ".yml") {
			continue
		}
		// 检查是否是允许恢复的文件
		baseName := strings.TrimSuffix(f.Name, ".yml")
		allowed := false
		for _, name := range backupFiles {
			if baseName == name {
				allowed = true
				break
			}
		}
		if !allowed {
			continue
		}

		// 读取zip中的文件内容
		rc, openErr := f.Open()
		if openErr != nil {
			continue
		}
		content, readErr := io.ReadAll(rc)
		rc.Close()
		if readErr != nil {
			continue
		}

		// 写入到工作目录
		targetPath := filepath.Join(workDir, f.Name)
		if writeErr := os.WriteFile(targetPath, content, os.ModePerm); writeErr != nil {
			continue
		}
		restoredCount++
	}

	// 返回成功页面
	return c.HTML(http.StatusOK, fmt.Sprintf(`<html><head><meta charset="UTF-8"><meta http-equiv="refresh" content="2;url=%s"></head><body><p>恢复成功，共恢复 %d 个配置文件，2秒后自动跳转...</p></body></html>`, define.SettingPages.Backup.Path, restoredCount))
}
