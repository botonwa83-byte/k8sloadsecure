package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"k8sgate/config"
	"k8sgate/middleware"
	"k8sgate/model"
	"k8sgate/pkg"
	"k8sgate/service"
)

// getRealIP 从请求中获取真实客户端IP
func getRealIP(c *gin.Context) string {
	// 优先从 X-Forwarded-For 获取
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		ip := strings.TrimSpace(parts[0])
		if ip != "" {
			return ip
		}
	}
	// 其次从 X-Real-IP 获取
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return xri
	}
	// 最后使用 Gin 默认方法
	return c.ClientIP()
}

type AuthHandler struct {
	cfg *config.Config
}

func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{cfg: cfg}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}

	user, token, err := service.Login(&req, getRealIP(c), h.cfg.PasswordMaxAge)
	if err != nil {
		pkg.Fail(c, http.StatusUnauthorized, 40101, err.Error())
		return
	}

	c.SetCookie("token", token, 8*3600, "/", "", false, true)

	pkg.OK(c, gin.H{
		"user_id":          user.ID,
		"username":         user.Username,
		"display_name":     user.DisplayName,
		"role":             user.Role,
		"password_expired": service.IsPasswordExpired(user),
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetCookie("token", "", -1, "/", "", false, true)
	pkg.OKMsg(c, "已登出")
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req service.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}

	userID := middleware.GetUserID(c)
	if err := service.ChangePassword(userID, &req, h.cfg.PasswordMaxAge); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, err.Error())
		return
	}

	c.SetCookie("token", "", -1, "/", "", false, true)
	pkg.OKMsg(c, "密码修改成功，请重新登录")
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var user model.User
	if err := model.DB.First(&user, userID).Error; err != nil {
		pkg.Fail(c, http.StatusUnauthorized, 40101, "用户不存在")
		return
	}

	// 获取用户项目
	var ups []model.UserProject
	model.DB.Where("user_id = ?", userID).Preload("Project.Namespaces").Find(&ups)

	type projectInfo struct {
		ProjectID   uint     `json:"project_id"`
		ProjectName string   `json:"project_name"`
		Permission  string   `json:"permission"`
		Namespaces  []string `json:"namespaces"`
	}
	projects := make([]projectInfo, 0, len(ups))
	for _, up := range ups {
		if up.Project == nil {
			continue
		}
		nsList := make([]string, len(up.Project.Namespaces))
		for i, ns := range up.Project.Namespaces {
			nsList[i] = ns.Namespace
		}
		projects = append(projects, projectInfo{
			ProjectID:   up.ProjectID,
			ProjectName: up.Project.Name,
			Permission:  up.Permission,
			Namespaces:  nsList,
		})
	}

	pkg.OK(c, gin.H{
		"user_id":            user.ID,
		"username":           user.Username,
		"display_name":       user.DisplayName,
		"email":              user.Email,
		"role":               user.Role,
		"projects":           projects,
		"password_expires_at": user.PasswordExpiresAt,
	})
}
