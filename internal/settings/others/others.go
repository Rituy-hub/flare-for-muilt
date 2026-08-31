package others

import (
	"strings"
	"log"
	"html/template"
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/soulteary/flare/config/data"
	"github.com/soulteary/flare/config/define"
	"github.com/soulteary/flare/internal/auth"
	"github.com/soulteary/flare/internal/pool"
	version "github.com/soulteary/version-kit"
)

func RegisterRouting(e *echo.Echo) {
	e.GET(define.SettingPages.Others.Path, pageOthers)
	e.POST(define.SettingPages.Others.Path, updateOthers)
}

func pageOthers(c *echo.Context) error {
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
	m["PageName"] = "Others"
	m["SettingPages"] = define.SettingPages
	m["OptionShowMultiUser"] = options.ShowMultiUser
	m["OptionTitle"] = options.Title
	// 传递管理员权限状态
	currentUser := auth.GetUserName(c)
	m["IsAdmin"] = currentUser == "" || auth.IsAdminUser(currentUser)
	m["Version"] = version.Version
	m["BuildDate"] = version.BuildDate
	m["COMMIT"] = version.Commit
	m["OptionFooter"] = template.HTML(options.Footer)
	return c.Render(http.StatusOK, "settings.html", m)
}

// updateOthers 处理其他设置页面的POST请求（修改用户名等）
func updateOthers(c *echo.Context) error {
	action := c.FormValue("action")

	if action == "change_username" {
		// 只有管理员才能修改用户名
		username := auth.GetUserName(c)
		if username == "" || !auth.IsAdminUser(username) {
			return c.Redirect(http.StatusFound, define.SettingPages.Others.Path)
		}

		// 验证当前密码
		currentPassword := c.FormValue("current_password")
		if !auth.VerifyUser(username, currentPassword) {
			log.Printf("[设置] 修改用户名失败: 当前密码验证不通过, username=%s", username)
			return c.Redirect(http.StatusFound, define.SettingPages.Others.Path)
		}

		newUsername := strings.TrimSpace(c.FormValue("new_username"))
		if newUsername == "" {
			return c.Redirect(http.StatusFound, define.SettingPages.Others.Path)
		}

		// 检查新用户名是否已存在
		if exists, _ := auth.UserExists(newUsername); exists {
			return c.Redirect(http.StatusFound, define.SettingPages.Others.Path)
		}

		// 修改管理员用户名
		if err := auth.UpdateAdminUsername(username, newUsername); err != nil {
			log.Printf("[设置] 修改管理员用户名失败: %v", err)
			return c.Redirect(http.StatusFound, define.SettingPages.Others.Path)
		}

		log.Printf("[设置] 管理员用户名已修改: %s -> %s", username, newUsername)
		return c.Redirect(http.StatusFound, define.SettingPages.Others.Path)
	}

	return c.Redirect(http.StatusFound, define.SettingPages.Others.Path)
}
