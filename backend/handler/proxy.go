package handler

import (
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
	"k8sgate/config"
	"k8sgate/k8s"
	"k8sgate/middleware"
	"k8sgate/model"
	"k8sgate/pkg"
)

type ProxyHandler struct {
	cfg *config.Config
}

func NewProxyHandler(cfg *config.Config) *ProxyHandler {
	return &ProxyHandler{cfg: cfg}
}

func (h *ProxyHandler) Proxy(c *gin.Context) {
	userID := middleware.GetUserID(c)
	role := middleware.GetRole(c)

	// 获取用户可访问的命名空间
	allowedNS, err := h.getAllowedNamespaces(userID, role)
	if err != nil {
		pkg.Fail(c, http.StatusInternalServerError, 50001, "获取权限失败")
		return
	}

	// 获取对应的 ServiceAccount Token
	token, err := k8s.GetTokenForUser(userID, role)
	if err != nil {
		pkg.Fail(c, http.StatusInternalServerError, 50002, "获取K8s凭证失败")
		return
	}

	// 前置权限检查（非 Admin）
	if role != "admin" {
		ns := extractNamespace(c.Request.URL.Path)

		// developer: 只能访问分配的命名空间
		if role == "developer" {
			if ns != "" && !contains(allowedNS, ns) {
				pkg.Fail(c, http.StatusForbidden, 40301, "无权访问命名空间: "+ns)
				return
			}
			// developer 默认只读，只有审批通过的项目才有写权限
			if c.Request.Method != "GET" && c.Request.Method != "HEAD" {
				if !h.hasWritePermission(userID, ns) {
					pkg.Fail(c, http.StatusForbidden, 40301, "无写权限，请先申请并等待管理员审批")
					return
				}
			}
		}

		// global_viewer: 可访问所有命名空间，但只能读
		if role == "global_viewer" {
			if c.Request.Method != "GET" && c.Request.Method != "HEAD" {
				pkg.Fail(c, http.StatusForbidden, 40301, "全局只读用户不允许写操作")
				return
			}
		}
	}

	target, _ := url.Parse(h.cfg.DashboardURL)
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// 注入 ServiceAccount Token
	c.Request.Header.Set("Authorization", "Bearer "+token)
	// 去掉 /dashboard 前缀
	c.Request.URL.Path = c.Param("path")
	if c.Request.URL.Path == "" {
		c.Request.URL.Path = "/"
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}

func (h *ProxyHandler) getAllowedNamespaces(userID uint, role string) ([]string, error) {
	// admin 和 global_viewer 无限制
	if role == "admin" || role == "global_viewer" {
		return nil, nil
	}

	// developer: 只能访问分配项目的命名空间
	var ups []model.UserProject
	err := model.DB.Where("user_id = ?", userID).Preload("Project.Namespaces").Find(&ups).Error
	if err != nil {
		return nil, err
	}

	nsSet := map[string]bool{}
	for _, up := range ups {
		if up.Project == nil {
			continue
		}
		for _, pn := range up.Project.Namespaces {
			nsSet[pn.Namespace] = true
		}
	}

	result := make([]string, 0, len(nsSet))
	for ns := range nsSet {
		result = append(result, ns)
	}
	return result, nil
}

// hasWritePermission 检查 developer 对某个命名空间是否有写权限（审批通过的）
func (h *ProxyHandler) hasWritePermission(userID uint, namespace string) bool {
	if namespace == "" {
		return false
	}
	var count int64
	model.DB.Model(&model.UserProject{}).
		Joins("JOIN project_namespaces ON project_namespaces.project_id = user_projects.project_id").
		Where("user_projects.user_id = ? AND project_namespaces.namespace = ? AND user_projects.permission = 'readwrite'", userID, namespace).
		Count(&count)
	return count > 0
}

func extractNamespace(path string) string {
	parts := splitPath(path)
	for i, p := range parts {
		if p == "namespaces" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func splitPath(path string) []string {
	result := []string{}
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
