package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"k8sgate/pkg"
)

const (
	CtxUserID   = "user_id"
	CtxUsername = "username"
	CtxRole    = "role"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr, err := c.Cookie("token")
		if err != nil || tokenStr == "" {
			pkg.Fail(c, http.StatusUnauthorized, 40102, "未登录或Token已过期")
			c.Abort()
			return
		}

		claims, err := pkg.ParseToken(tokenStr)
		if err != nil {
			pkg.Fail(c, http.StatusUnauthorized, 40102, "Token无效或已过期")
			c.Abort()
			return
		}

		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxUsername, claims.Username)
		c.Set(CtxRole, claims.Role)
		c.Next()
	}
}

func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get(CtxRole)
		if !exists || role.(string) != "admin" {
			pkg.Fail(c, http.StatusForbidden, 40301, "权限不足，需要管理员权限")
			c.Abort()
			return
		}
		c.Next()
	}
}

func GetUserID(c *gin.Context) uint {
	v, _ := c.Get(CtxUserID)
	if id, ok := v.(uint); ok {
		return id
	}
	return 0
}

func GetUsername(c *gin.Context) string {
	v, _ := c.Get(CtxUsername)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func GetRole(c *gin.Context) string {
	v, _ := c.Get(CtxRole)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
