package server

import (
	"errors"
	"net/http"
	"net/url"
	"sort"

	"github.com/gin-gonic/gin"

	"isrvd/internal/service/account"
)

// defineAccountRoutes 定义 Account 模块路由
func (app *App) defineAccountRoutes() []Route {
	return []Route{
		// 认证与登录
		{Method: "GET", Path: "/account/info", Handler: app.accountAuth, Module: "account", Label: "获取当前认证信息", Access: AccessAnon},
		{Method: "POST", Path: "/account/login", Handler: app.accountLogin, Module: "account", Label: "账号密码登录", Access: AccessAnon},
		// Passkey 登录（无需认证）
		{Method: "POST", Path: "/account/passkey/login/begin", Handler: app.accountPasskeyLoginBegin, Module: "account", Label: "开始 Passkey 登录", Access: AccessAnon},
		{Method: "POST", Path: "/account/passkey/login/finish", Handler: app.accountPasskeyLoginFinish, Module: "account", Label: "完成 Passkey 登录", Access: AccessAnon},
		// Passkey 注册/绑定（需要认证）
		{Method: "POST", Path: "/account/passkey/register/begin", Handler: app.accountPasskeyRegisterBegin, Module: "account", Label: "开始 Passkey 绑定", Access: AccessAuth},
		{Method: "POST", Path: "/account/passkey/register/finish", Handler: app.accountPasskeyRegisterFinish, Module: "account", Label: "完成 Passkey 绑定", Access: AccessAuth},
		{Method: "GET", Path: "/account/passkey/credentials", Handler: app.accountPasskeyListCredentials, Module: "account", Label: "查询 Passkey 凭证列表", Access: AccessAuth},
		{Method: "DELETE", Path: "/account/passkey/credential/:credentialID", Handler: app.accountPasskeyDeleteCredential, Module: "account", Label: "删除 Passkey 凭证", Access: AccessAuth},
		// OIDC 登录
		{Method: "GET", Path: "/account/oidc/login", Handler: app.accountOIDCLogin, Module: "account", Label: "发起 OIDC 登录", Access: AccessAnon},
		{Method: "GET", Path: "/account/oidc/callback", Handler: app.accountOIDCCallback, Module: "account", Label: "处理 OIDC 回调", Access: AccessAnon},
		{Method: "POST", Path: "/account/oidc/exchange", Handler: app.accountOIDCExchange, Module: "account", Label: "交换 OIDC 登录码", Access: AccessAnon},
		// 凭证管理
		{Method: "POST", Path: "/account/token", Handler: app.accountTokenCreate, Module: "account", Label: "创建 API 令牌"},
		{Method: "PUT", Path: "/account/password", Handler: app.accountPasswordChange, Module: "account", Label: "修改当前用户密码", Access: AccessAuth},
		// 路由权限
		{Method: "GET", Path: "/account/routes", Handler: app.accountRouteList, Module: "account", Label: "查询路由权限列表", Access: AccessAuth},
		// 成员管理
		{Method: "GET", Path: "/account/members", Handler: app.accountMemberList, Module: "account", Label: "查询成员列表"},
		{Method: "POST", Path: "/account/member", Handler: app.accountMemberCreate, Module: "account", Label: "创建成员"},
		{Method: "PUT", Path: "/account/member/:username", Handler: app.accountMemberUpdate, Module: "account", Label: "更新成员"},
		{Method: "DELETE", Path: "/account/member/:username", Handler: app.accountMemberDelete, Module: "account", Label: "删除成员"},
	}
}

// accountAuth 返回当前认证模式及已登录用户信息
func (app *App) accountAuth(c *gin.Context) {
	username := c.GetString("username")
	respondSuccess(c, "ok", app.accountSvc.AuthInfo(username))
}

// accountLogin 校验用户名密码并签发 JWT Token
func (app *App) accountLogin(c *gin.Context) {
	var req account.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := app.accountSvc.Login(req)
	if err != nil {
		respondError(c, http.StatusUnauthorized, err.Error())
		return
	}
	respondSuccess(c, "登录成功", resp)
}

// accountPasskeyRegisterBegin 开始 Passkey 注册
func (app *App) accountPasskeyRegisterBegin(c *gin.Context) {
	var req account.PasskeyBeginRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := app.accountSvc.PasskeyBeginRegistration(c, req)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "开始注册", resp)
}

// accountPasskeyRegisterFinish 完成 Passkey 注册
func (app *App) accountPasskeyRegisterFinish(c *gin.Context) {
	var req account.PasskeyFinishRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	err := app.accountSvc.PasskeyFinishRegistration(c, req)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "Passkey 注册成功", nil)
}

// accountPasskeyLoginBegin 开始 Passkey 登录
func (app *App) accountPasskeyLoginBegin(c *gin.Context) {
	var req account.PasskeyBeginLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := app.accountSvc.PasskeyBeginLogin(c, req)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "开始登录", resp)
}

// accountPasskeyLoginFinish 完成 Passkey 登录
func (app *App) accountPasskeyLoginFinish(c *gin.Context) {
	var req account.PasskeyFinishLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := app.accountSvc.PasskeyFinishLogin(c, req)
	if err != nil {
		respondError(c, http.StatusUnauthorized, err.Error())
		return
	}
	respondSuccess(c, "登录成功", resp)
}

// accountOIDCLogin 跳转到 OIDC Provider 登录页
func (app *App) accountOIDCLogin(c *gin.Context) {
	loginURL, err := app.accountSvc.OIDCLoginURL(c)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.Redirect(http.StatusFound, loginURL)
}

// accountOIDCCallback 处理 OIDC Provider 回调
func (app *App) accountOIDCCallback(c *gin.Context) {
	code, err := app.accountSvc.OIDCCallback(c)
	if err != nil {
		c.Redirect(http.StatusFound, "/?oidc_error="+url.QueryEscape(err.Error()))
		return
	}
	c.Redirect(http.StatusFound, "/?oidc_code="+url.QueryEscape(code))
}

// accountOIDCExchange 使用一次性登录码换取 JWT Token
func (app *App) accountOIDCExchange(c *gin.Context) {
	var req account.OIDCExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := app.accountSvc.OIDCExchange(req.Code)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "登录成功", resp)
}

// accountTokenCreate 创建长效 API Token
func (app *App) accountTokenCreate(c *gin.Context) {
	var req account.CreateApiTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	username := c.GetString("username")
	resp, err := app.accountSvc.ApiTokenCreate(username, req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "令牌创建成功", resp)
}

// accountPasswordChange 修改当前用户密码
func (app *App) accountPasswordChange(c *gin.Context) {
	var req account.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	username := c.GetString("username")
	if err := app.accountSvc.PasswordChange(username, req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	respondSuccess(c, "密码修改成功", nil)
}

// accountRouteList 返回所有已注册路由及其权限元信息
func (app *App) accountRouteList(c *gin.Context) {
	routes := make([]Route, 0, len(app.routeIndex))
	for _, route := range app.routeIndex {
		routes = append(routes, route)
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Module != routes[j].Module {
			return routes[i].Module < routes[j].Module
		}
		return routes[i].Key < routes[j].Key
	})
	respondSuccess(c, "ok", routes)
}

// accountMemberList 列出所有成员
func (app *App) accountMemberList(c *gin.Context) {
	respondSuccess(c, "ok", app.accountSvc.MemberList())
}

// accountMemberCreate 新建成员
func (app *App) accountMemberCreate(c *gin.Context) {
	var req account.MemberUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.accountSvc.MemberCreate(req); err != nil {
		switch {
		case errors.Is(err, account.ErrInvalidRequest), errors.Is(err, account.ErrMemberExists):
			respondError(c, http.StatusBadRequest, err.Error())
		default:
			respondError(c, http.StatusInternalServerError, err.Error())
		}
		return
	}
	respondSuccess(c, "成员添加成功", nil)
}

// accountMemberUpdate 更新成员
func (app *App) accountMemberUpdate(c *gin.Context) {
	username := c.Param("username")
	var req account.MemberUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := app.accountSvc.MemberUpdate(username, req); err != nil {
		if errors.Is(err, account.ErrMemberNotFound) {
			respondError(c, http.StatusNotFound, err.Error())
		} else {
			respondError(c, http.StatusInternalServerError, err.Error())
		}
		return
	}
	respondSuccess(c, "成员更新成功", nil)
}

// accountMemberDelete 删除成员
func (app *App) accountMemberDelete(c *gin.Context) {
	username := c.Param("username")
	if err := app.accountSvc.MemberDelete(username); err != nil {
		switch {
		case errors.Is(err, account.ErrMemberNotFound):
			respondError(c, http.StatusNotFound, err.Error())
		default:
			respondError(c, http.StatusInternalServerError, err.Error())
		}
		return
	}
	respondSuccess(c, "成员删除成功", nil)
}

// accountPasskeyListCredentials 查询当前用户的 Passkey 凭证列表
func (app *App) accountPasskeyListCredentials(c *gin.Context) {
	username := c.GetString("username")
	credentials, err := app.accountSvc.PasskeyListCredentials(username)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondSuccess(c, "查询成功", credentials)
}

// accountPasskeyDeleteCredential 删除当前用户的指定 Passkey 凭证
func (app *App) accountPasskeyDeleteCredential(c *gin.Context) {
	username := c.GetString("username")
	credentialID := c.Param("credentialID")
	if err := app.accountSvc.PasskeyDeleteCredential(username, credentialID); err != nil {
		switch {
		case errors.Is(err, account.ErrPasskeyNotFound):
			respondError(c, http.StatusNotFound, err.Error())
		default:
			respondError(c, http.StatusInternalServerError, err.Error())
		}
		return
	}
	respondSuccess(c, "凭证删除成功", nil)
}
