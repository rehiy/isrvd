// Package system 成员账号管理
package system

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/rehiy/pango/logman"

	"isrvd/config"
	"isrvd/internal/helper"
)

// MemberInfo 成员信息（GET 时返回，不包含密码明文）
type MemberInfo struct {
	Username      string `json:"username"`
	HomeDirectory string `json:"homeDirectory"`
	AllowTerminal bool   `json:"allowTerminal"`
	PasswordSet   bool   `json:"passwordSet"`
	IsPrimary     bool   `json:"isPrimary"`
}

// MemberUpsertRequest 成员新建/更新请求
type MemberUpsertRequest struct {
	Username      string `json:"username"`
	Password      string `json:"password"`
	HomeDirectory string `json:"homeDirectory"`
	AllowTerminal bool   `json:"allowTerminal"`
}

// ListMembers 列出所有成员
func (h *SettingsHandler) ListMembers(c *gin.Context) {
	list := make([]*MemberInfo, 0, len(config.Members))
	for _, m := range config.Members {
		list = append(list, &MemberInfo{
			Username:      m.Username,
			HomeDirectory: m.HomeDirectory,
			AllowTerminal: m.AllowTerminal,
			PasswordSet:   m.Password != "",
			IsPrimary:     m.Username == config.PrimaryMember,
		})
	}
	helper.RespondSuccess(c, "ok", list)
}

// ensureHomeDir 生成并创建成员 home 目录（空值时使用基础目录 + 用户名）
func ensureHomeDir(home, username string) (string, error) {
	if home == "" {
		home = username
	}
	if !filepath.IsAbs(home) {
		home = filepath.Join(config.RootDirectory, home)
	}
	if err := os.MkdirAll(home, 0755); err != nil {
		return "", err
	}
	return home, nil
}

// CreateMember 新建成员
func (h *SettingsHandler) CreateMember(c *gin.Context) {
	var req MemberUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的JSON")
		return
	}
	if req.Username == "" {
		helper.RespondError(c, http.StatusBadRequest, "用户名不能为空")
		return
	}
	if req.Password == "" {
		helper.RespondError(c, http.StatusBadRequest, "密码不能为空")
		return
	}
	if _, exists := config.Members[req.Username]; exists {
		helper.RespondError(c, http.StatusBadRequest, "用户名已存在")
		return
	}

	home, err := ensureHomeDir(req.HomeDirectory, req.Username)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "创建 home 目录失败: "+err.Error())
		return
	}

	config.Members[req.Username] = &config.MemberConfig{
		Username:      req.Username,
		Password:      req.Password,
		HomeDirectory: home,
		AllowTerminal: req.AllowTerminal,
	}
	// 首个成员设为主账号（不可删除）
	if config.PrimaryMember == "" {
		config.PrimaryMember = req.Username
	}
	if err := config.Save(); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "保存配置失败: "+err.Error())
		return
	}
	logman.Info("Member created", "username", req.Username)
	helper.RespondSuccess(c, "成员添加成功", nil)
}

// UpdateMember 更新成员
func (h *SettingsHandler) UpdateMember(c *gin.Context) {
	username := c.Param("username")
	member, exists := config.Members[username]
	if !exists {
		helper.RespondError(c, http.StatusNotFound, "成员不存在")
		return
	}
	var req MemberUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.RespondError(c, http.StatusBadRequest, "无效的JSON")
		return
	}

	home, err := ensureHomeDir(req.HomeDirectory, username)
	if err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "创建 home 目录失败: "+err.Error())
		return
	}

	member.Password = pickSecret(req.Password, member.Password)
	member.HomeDirectory = home
	member.AllowTerminal = req.AllowTerminal

	if err := config.Save(); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "保存配置失败: "+err.Error())
		return
	}
	logman.Info("Member updated", "username", username)
	helper.RespondSuccess(c, "成员更新成功", nil)
}

// DeleteMember 删除成员
func (h *SettingsHandler) DeleteMember(c *gin.Context) {
	username := c.Param("username")
	if _, exists := config.Members[username]; !exists {
		helper.RespondError(c, http.StatusNotFound, "成员不存在")
		return
	}
	if username == config.PrimaryMember {
		helper.RespondError(c, http.StatusForbidden, "主账号禁止删除")
		return
	}
	delete(config.Members, username)
	if err := config.Save(); err != nil {
		helper.RespondError(c, http.StatusInternalServerError, "保存配置失败: "+err.Error())
		return
	}
	logman.Info("Member deleted", "username", username)
	helper.RespondSuccess(c, "成员删除成功", nil)
}
