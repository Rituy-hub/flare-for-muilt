package auth

import (
	"crypto/subtle"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	session "github.com/labstack/echo-contrib/v5/session"
	"github.com/labstack/echo/v5"

	"github.com/soulteary/flare/config/data"
	"github.com/soulteary/flare/config/define"
)

const (
	SESSION_KEY_USER_NAME  = "USER_NAME"
	SESSION_KEY_LOGIN_DATE = "LOGIN_TIME"
)

// sessionName is set by RequestHandle and used by session.Get. Prefer passing name via RequestHandleSessionName.
var sessionName string

// RequestHandleSessionName returns the session name for the given cookie name and port (for testing or explicit wiring).
func RequestHandleSessionName(cookieName string, port int) string {
	return fmt.Sprintf("%s_%d", cookieName, port)
}

func RequestHandle(e *echo.Echo) {
	// fallback: 如果 CookieName 或 CookieSecret 为空（环境变量空值覆盖了默认值），使用默认值
	cookieName := define.AppFlags.CookieName
	if cookieName == "" {
		cookieName = define.DEFAULT_COOKIE_NAME
	}
	cookieSecret := define.AppFlags.CookieSecret
	if cookieSecret == "" {
		cookieSecret = define.DEFAULT_COOKIE_SECRET
	}
	sessionName = RequestHandleSessionName(cookieName, define.AppFlags.Port)
	if !define.AppFlags.DisableLoginMode {
		if cookieSecret == define.DEFAULT_COOKIE_SECRET {
			log.Println("[auth] 警告: 已启用登录但 CookieSecret 仍为默认值，生产环境请通过 FLARE_COOKIE_SECRET 或 --cookie-secret 设置强密钥")
		}
		store := sessions.NewCookieStore([]byte(cookieSecret))
		// 明确设置 Cookie 选项，避免默认值在某些环境下导致 Cookie 无法写入
		store.Options = &sessions.Options{
			Path:     "/",
			Domain:   "",
			MaxAge:   86400 * 7, // 7天
			Secure:   false,
			HttpOnly: true,
		}
		e.Use(session.Middleware(store))
		e.GET(define.MiscPages.Login.Path, loginPage)
		e.POST(define.MiscPages.Login.Path, login)
		e.POST(define.MiscPages.Logout.Path, logout)
		e.GET(define.MiscPages.Register.Path, registerPage)
		e.POST(define.MiscPages.Register.Path, register)
		e.GET("/change-password", changePasswordPage)
		e.POST("/change-password", changePassword)
		log.Printf("[auth] 登录模式已启用，session名称=%s，cookie密钥长度=%d", sessionName, len(cookieSecret))
	}
}

var commonText = `<a href="` + define.SettingPages.Others.Path + `">返回重试</a></p><p>或前往 <a href="https://github.com/soulteary/docker-flare/issues/" target="_blank">https://github.com/soulteary/docker-flare/issues/</a> 反馈使用中的问题，谢谢！`
var internalErrorInput = []byte(`<html><p>请填写正确的用户名和密码 ` + commonText + `</html>`)
var internalErrorEmpty = []byte(`<html><p>用户名或密码不能为空 ` + commonText + `</html>`)
var internalErrorSave = []byte(`<html><p>程序内部错误，保存登陆状态失败 ` + commonText + `</html>`)

func AuthRequired(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if !define.AppFlags.DisableLoginMode {
			sess, err := session.Get(sessionName, c)
			if err != nil {
				return c.Redirect(http.StatusFound, define.SettingPages.Others.Path)
			}
			user := sess.Values[SESSION_KEY_USER_NAME]
			if user == nil {
				return c.Redirect(http.StatusFound, define.SettingPages.Others.Path)
			}
		}
		return next(c)
	}
}

func CheckUserIsLogin(c *echo.Context) bool {
	if !define.AppFlags.DisableLoginMode {
		sess, err := session.Get(sessionName, c)
		if err != nil {
			return false
		}
		user := sess.Values[SESSION_KEY_USER_NAME]
		return user != nil
	}
	return true
}

func GetUserName(c *echo.Context) string {
	if !define.AppFlags.DisableLoginMode {
		sess, err := session.Get(sessionName, c)
		if err != nil {
			return ""
		}
		if v, ok := sess.Values[SESSION_KEY_USER_NAME].(string); ok {
			return v
		}
	}
	return ""
}

func GetUserLoginDate(c *echo.Context) string {
	if !define.AppFlags.DisableLoginMode {
		sess, err := session.Get(sessionName, c)
		if err != nil {
			return ""
		}
		if v, ok := sess.Values[SESSION_KEY_LOGIN_DATE].(string); ok {
			return v
		}
	}
	return ""
}

// changePasswordPage 修改密码页面
func changePasswordPage(c *echo.Context) error {
	username := GetUserName(c)
	if username == "" {
		return c.Redirect(http.StatusFound, define.MiscPages.Login.Path)
	}

	html := `<!doctype html><html lang=zh><head><meta charset=UTF-8><meta name=viewport content="width=device-width,initial-scale=1"><title>修改密码</title><style>
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;background:#f5f5f5;display:flex;justify-content:center;align-items:center;min-height:100vh;margin:0}
.login-box{background:#fff;padding:40px;border-radius:8px;box-shadow:0 2px 10px rgba(0,0,0,.1);width:360px}
.login-box h2{text-align:center;margin:0 0 10px;color:#333}
.login-box .tip{text-align:center;color:#e74c3c;font-size:14px;margin-bottom:20px}
.form-group{margin-bottom:20px}
.form-group label{display:block;margin-bottom:8px;color:#666;font-size:14px}
.form-group input{width:100%;padding:10px;border:1px solid #ddd;border-radius:4px;font-size:14px;box-sizing:border-box}
.form-group input:focus{outline:none;border-color:#4a90d9}
.btn-login{width:100%;padding:12px;background:#4a90d9;color:#fff;border:none;border-radius:4px;font-size:16px;cursor:pointer}
.btn-login:hover{background:#357abd}
.error{color:#e74c3c;font-size:14px;margin-bottom:15px;text-align:center}
</style></head><body><div class=login-box><h2>修改密码</h2><p class=tip>首次登录请修改密码</p>
<form method=POST action=/change-password>
<div class=form-group><label>当前用户：` + username + `</label></div>
<div class=form-group><label>新密码</label><input type=password name=new_password placeholder="请输入新密码" required></div>
<div class=form-group><label>确认新密码</label><input type=password name=confirm_password placeholder="请再次输入新密码" required></div>
<button type=submit class=btn-login>确认修改</button>
</form>
</div></body></html>`
	return c.HTML(http.StatusOK, html)
}

// changePassword 修改密码处理
func changePassword(c *echo.Context) error {
	username := GetUserName(c)
	if username == "" {
		return c.Redirect(http.StatusFound, define.MiscPages.Login.Path)
	}

	newPassword := c.FormValue("new_password")
	confirmPassword := c.FormValue("confirm_password")

	renderError := func(msg string) string {
		return `<!doctype html><html lang=zh><head><meta charset=UTF-8><meta name=viewport content="width=device-width,initial-scale=1"><title>修改密码</title><style>
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;background:#f5f5f5;display:flex;justify-content:center;align-items:center;min-height:100vh;margin:0}
.login-box{background:#fff;padding:40px;border-radius:8px;box-shadow:0 2px 10px rgba(0,0,0,.1);width:360px}
.error{color:#e74c3c;font-size:14px;margin-bottom:15px;text-align:center}
.btn-back{display:block;text-align:center;padding:12px;background:#4a90d9;color:#fff;text-decoration:none;border-radius:4px;font-size:16px}
</style></head><body><div class=login-box><p class=error>` + msg + `</p><a href=/change-password class=btn-back>返回修改</a></div></body></html>`
	}

	if newPassword == "" || confirmPassword == "" {
		return c.HTML(http.StatusBadRequest, renderError("密码不能为空"))
	}

	if newPassword != confirmPassword {
		return c.HTML(http.StatusBadRequest, renderError("两次输入的密码不一致"))
	}

	if err := ChangeUserPassword(username, newPassword); err != nil {
		return c.HTML(http.StatusBadRequest, renderError("修改密码失败: "+err.Error()))
	}

	log.Printf("[auth] 密码修改成功: username=%s", username)

	successHTML := `<!doctype html><html lang=zh><head><meta charset=UTF-8><meta name=viewport content="width=device-width,initial-scale=1"><title>修改成功</title><style>
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;background:#f5f5f5;display:flex;justify-content:center;align-items:center;min-height:100vh;margin:0}
.login-box{background:#fff;padding:40px;border-radius:8px;box-shadow:0 2px 10px rgba(0,0,0,.1);width:360px;text-align:center}
.success{color:#27ae60;font-size:18px;margin-bottom:20px}
.btn-login{display:block;padding:12px;background:#4a90d9;color:#fff;text-decoration:none;border-radius:4px;font-size:16px}
</style></head><body><div class=login-box><p class=success>密码修改成功！请重新登录</p><a href=` + define.MiscPages.Login.Path + ` class=btn-login>前往登录</a></div></body></html>`
	return c.HTML(http.StatusOK, successHTML)
}

// loginPage 登录页面
func loginPage(c *echo.Context) error {
	// 检查是否已登录
	sess, err := session.Get(sessionName, c)
	if err == nil && sess.Values[SESSION_KEY_USER_NAME] != nil {
		return c.Redirect(http.StatusFound, define.SettingPages.Others.Path)
	}

	// 检查是否启用多用户功能
	showRegister := false
	if options, err := data.GetAllSettingsOptions(); err == nil {
		showRegister = options.ShowMultiUser
	}

	registerBtn := ""
	if showRegister {
		registerBtn = `<button type=button class=btn-login style="flex:1;" onclick="location.href='` + define.MiscPages.Register.Path + `'">注册</button>`
	}

	html := `<!doctype html><html lang=zh><head><meta charset=UTF-8><meta name=viewport content="width=device-width,initial-scale=1"><title>登录</title><style>
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;background:#f5f5f5;display:flex;justify-content:center;align-items:center;min-height:100vh;margin:0}
.login-box{background:#fff;padding:40px;border-radius:8px;box-shadow:0 2px 10px rgba(0,0,0,.1);width:360px}
.login-box h2{text-align:center;margin:0 0 30px;color:#333}
.form-group{margin-bottom:20px}
.form-group label{display:block;margin-bottom:8px;color:#666;font-size:14px}
.form-group input{width:100%;padding:10px;border:1px solid #ddd;border-radius:4px;font-size:14px;box-sizing:border-box}
.form-group input:focus{outline:none;border-color:#4a90d9}
.btn-login{width:100%;padding:12px;background:#4a90d9;color:#fff;border:none;border-radius:4px;font-size:16px;cursor:pointer}
.btn-login:hover{background:#357abd}
.btn-register{width:100%;padding:12px;background:#4a90d9;color:#fff;border:none;border-radius:4px;font-size:16px;cursor:pointer}
.btn-register:hover{background:#357abd}
</style></head><body><div class=login-box><h2>用户登录</h2>
<form method=POST action=` + define.MiscPages.Login.Path + `>
<div class=form-group><label>用户名</label><input type=text name=username placeholder="请输入用户名" required></div>
<div class=form-group><label>密码</label><input type=password name=password placeholder="请输入密码" required></div>
<div style="display:flex;gap:10px;"><button type=submit class=btn-login style="flex:1;">登录</button>` + registerBtn + `</div>
</form>
</div></body></html>`
	return c.HTML(http.StatusOK, html)
}

func login(c *echo.Context) error {
	sess, err := session.Get(sessionName, c)
	if err != nil {
		log.Printf("[auth] 登录失败: session.Get 出错, sessionName=%s, error=%v", sessionName, err)
		return c.HTMLBlob(http.StatusBadRequest, internalErrorSave)
	}
	username := c.FormValue("username")
	password := c.FormValue("password")

	if strings.Trim(username, " ") == "" || strings.Trim(password, " ") == "" {
		return c.HTMLBlob(http.StatusBadRequest, internalErrorEmpty)
	}

	// 多用户登录：优先检查用户列表，然后 fallback 到管理员账户
	isValidUser := false
	if VerifyUser(username, password) {
		isValidUser = true
	} else if subtle.ConstantTimeCompare([]byte(username), []byte(define.AppFlags.User)) == 1 &&
		subtle.ConstantTimeCompare([]byte(password), []byte(define.AppFlags.Pass)) == 1 {
		isValidUser = true
	}

	if !isValidUser {
		log.Printf("[auth] 登录失败: 用户名或密码错误, username=%s", username)
		return c.HTMLBlob(http.StatusBadRequest, internalErrorInput)
	}

	// 初始化用户数据目录
	if err := InitUserDir(username); err != nil {
		log.Printf("[auth] 初始化用户目录失败: username=%s, error=%v", username, err)
	}

	sess.Values[SESSION_KEY_USER_NAME] = username
	sess.Values[SESSION_KEY_LOGIN_DATE] = time.Now().Format("2006年01月02日 15:04:05 CST")

	if err := sess.Save(c.Request(), c.Response()); err != nil {
		log.Printf("[auth] 登录失败: session.Save 出错, username=%s, error=%v", username, err)
		return c.HTMLBlob(http.StatusBadRequest, internalErrorSave)
	}

	log.Printf("[auth] 登录成功: username=%s", username)
	// 检查是否需要强制修改密码
	if MustChangePasswordUser(username) {
		return c.Redirect(http.StatusFound, "/change-password")
	}
	return c.Redirect(http.StatusFound, define.SettingPages.Others.Path)
}

func logout(c *echo.Context) error {
	sess, err := session.Get(sessionName, c)
	if err != nil {
		log.Printf("[auth] 登出失败: session.Get 出错, error=%v", err)
		return c.Redirect(http.StatusFound, define.SettingPages.Others.Path)
	}
	if sess.Values[SESSION_KEY_USER_NAME] == nil {
		return c.Redirect(http.StatusFound, define.SettingPages.Others.Path)
	}
	delete(sess.Values, SESSION_KEY_USER_NAME)
	delete(sess.Values, SESSION_KEY_LOGIN_DATE)

	if err := sess.Save(c.Request(), c.Response()); err != nil {
		log.Printf("[auth] 登出失败: session.Save 出错, error=%v", err)
		return c.HTMLBlob(http.StatusBadRequest, internalErrorSave)
	}
	return c.Redirect(http.StatusFound, define.SettingPages.Others.Path)
}

// registerPage 注册页面
func registerPage(c *echo.Context) error {
	html := `<!doctype html><html lang=zh><head><meta charset=UTF-8><meta name=viewport content="width=device-width,initial-scale=1"><title>用户注册</title><style>
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;background:#f5f5f5;display:flex;justify-content:center;align-items:center;min-height:100vh;margin:0}
.register-box{background:#fff;padding:40px;border-radius:8px;box-shadow:0 2px 10px rgba(0,0,0,.1);width:360px}
.register-box h2{text-align:center;margin:0 0 30px;color:#333}
.form-group{margin-bottom:20px}
.form-group label{display:block;margin-bottom:8px;color:#666;font-size:14px}
.form-group input{width:100%;padding:10px;border:1px solid #ddd;border-radius:4px;font-size:14px;box-sizing:border-box}
.form-group input:focus{outline:none;border-color:#4a90d9}
.btn-register{width:100%;padding:12px;background:#4a90d9;color:#fff;border:none;border-radius:4px;font-size:16px;cursor:pointer}
.btn-register:hover{background:#357abd}
.btn-login{width:100%;padding:12px;background:#f0f0f0;color:#333;border:none;border-radius:4px;font-size:16px;cursor:pointer;margin-top:10px}
.btn-login:hover{background:#e0e0e0}
</style></head><body><div class=register-box><h2>用户注册</h2>
<form method=POST action=` + define.MiscPages.Register.Path + `>
<div class=form-group><label>用户名</label><input type=text name=username placeholder="请输入用户名" required></div>
<div class=form-group><label>请设置密码</label><input type=password name=password placeholder="请设置密码" required></div>
<div class=form-group><label>请再次确认密码</label><input type=password name=password2 placeholder="请再次确认密码" required></div>
<button type=submit class=btn-register>注册</button>
</form>
<form method=GET action=` + define.MiscPages.Login.Path + `><button type=submit class=btn-login>返回登录</button></form>
</div></body></html>`
	return c.HTML(http.StatusOK, html)
}

// register 注册处理
func register(c *echo.Context) error {
	username := strings.TrimSpace(c.FormValue("username"))
	password := c.FormValue("password")
	password2 := c.FormValue("password2")

	renderError := func(msg string) string {
		return `<!doctype html><html lang=zh><head><meta charset=UTF-8><meta name=viewport content="width=device-width,initial-scale=1"><title>用户注册</title><style>
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;background:#f5f5f5;display:flex;justify-content:center;align-items:center;min-height:100vh;margin:0}
.register-box{background:#fff;padding:40px;border-radius:8px;box-shadow:0 2px 10px rgba(0,0,0,.1);width:360px}
.register-box h2{text-align:center;margin:0 0 30px;color:#333}
.error{color:#e74c3c;font-size:14px;margin-bottom:15px;text-align:center}
.btn-back{display:block;text-align:center;padding:12px;background:#4a90d9;color:#fff;text-decoration:none;border-radius:4px;font-size:16px}
</style></head><body><div class=register-box><h2>用户注册</h2><p class=error>` + msg + `</p><a href=` + define.MiscPages.Register.Path + ` class=btn-back>返回注册</a></div></body></html>`
	}

	if username == "" || password == "" {
		return c.HTML(http.StatusBadRequest, renderError("用户名或密码不能为空"))
	}

	if password != password2 {
		return c.HTML(http.StatusBadRequest, renderError("两次输入的密码不一致"))
	}

	if err := RegisterUser(username, password); err != nil {
		return c.HTML(http.StatusBadRequest, renderError(err.Error()))
	}

	if err := InitUserDir(username); err != nil {
		log.Printf("[auth] 初始化用户目录失败: username=%s, error=%v", username, err)
	}

	log.Printf("[auth] 注册成功: username=%s", username)

	successHTML := `<!doctype html><html lang=zh><head><meta charset=UTF-8><meta name=viewport content="width=device-width,initial-scale=1"><title>注册成功</title><style>
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;background:#f5f5f5;display:flex;justify-content:center;align-items:center;min-height:100vh;margin:0}
.register-box{background:#fff;padding:40px;border-radius:8px;box-shadow:0 2px 10px rgba(0,0,0,.1);width:360px;text-align:center}
.success{color:#27ae60;font-size:18px;margin-bottom:20px}
.btn-login{display:block;padding:12px;background:#4a90d9;color:#fff;text-decoration:none;border-radius:4px;font-size:16px}
</style></head><body><div class=register-box><p class=success>注册成功！请使用新账号登录</p><a href=` + define.MiscPages.Login.Path + ` class=btn-login>前往登录</a></div></body></html>`
	return c.HTML(http.StatusOK, successHTML)
}
