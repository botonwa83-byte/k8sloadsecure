package handler

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"k8sgate/config"
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

func (h *ProxyHandler) Status(c *gin.Context) {
	target, err := url.Parse(h.cfg.DashboardURL)
	if err != nil {
		pkg.OK(c, gin.H{"available": false, "reason": "Dashboard URL 配置错误: " + err.Error()})
		return
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Get(target.String())
	if err != nil {
		pkg.OK(c, gin.H{"available": false, "reason": "无法连接 Dashboard 服务: " + err.Error()})
		return
	}
	resp.Body.Close()

	pkg.OK(c, gin.H{"available": true, "url": h.cfg.DashboardURL})
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
				if ns != "" {
					if !h.hasWritePermission(userID, ns) {
						pkg.Fail(c, http.StatusForbidden, 40301, "无写权限，请先申请并等待管理员审批")
						return
					}
				} else {
					// 无法从路径解析命名空间时，检查用户是否拥有任意命名空间的写权限
					if !h.hasAnyWritePermission(userID) {
						pkg.Fail(c, http.StatusForbidden, 40301, "无写权限，请先申请并等待管理员审批")
						return
					}
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

	// 对非 admin 用户，拦截命名空间列表响应并过滤
	if role != "admin" && allowedNS != nil {
		proxyPath := c.Param("path")
		if isNamespaceListRequest(c.Request.Method, proxyPath) {
			proxy.ModifyResponse = h.namespaceFilterResponse(allowedNS)
		}
	}

	// 不注入 Authorization 头 — Dashboard v2 有自己的 session 管理，
	// 注入外部 token 会破坏其 CSRF/session 机制。
	// Dashboard 使用自己的 SA（已绑定 cluster-admin）访问 K8s API，
	// 访问控制由 K8sGate proxy 层（上面的权限检查）负责。
	c.Request.Header.Del("Authorization")

	// 去掉 /dashboard 前缀
	c.Request.URL.Path = c.Param("path")
	if c.Request.URL.Path == "" {
		c.Request.URL.Path = "/"
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}

// MyNamespaces 返回当前用户可访问的命名空间列表
func (h *ProxyHandler) MyNamespaces(c *gin.Context) {
	userID := middleware.GetUserID(c)
	role := middleware.GetRole(c)

	if role == "admin" || role == "global_viewer" {
		// admin 和 global_viewer 返回所有命名空间
		namespaces, err := getAllNamespacesFromDB()
		if err != nil {
			pkg.Fail(c, http.StatusInternalServerError, 50001, "获取命名空间失败")
			return
		}
		pkg.OK(c, gin.H{"namespaces": namespaces, "all": true})
		return
	}

	// developer: 只返回分配的命名空间
	allowedNS, err := h.getAllowedNamespaces(userID, role)
	if err != nil {
		pkg.Fail(c, http.StatusInternalServerError, 50001, "获取权限失败")
		return
	}
	pkg.OK(c, gin.H{"namespaces": allowedNS, "all": false})
}

// getAllNamespacesFromDB 从数据库获取所有已分配的命名空间
func getAllNamespacesFromDB() ([]string, error) {
	var pns []model.ProjectNamespace
	err := model.DB.Find(&pns).Error
	if err != nil {
		return nil, err
	}
	nsSet := map[string]bool{}
	for _, pn := range pns {
		nsSet[pn.Namespace] = true
	}
	result := make([]string, 0, len(nsSet))
	for ns := range nsSet {
		result = append(result, ns)
	}
	return result, nil
}

// isNamespaceListRequest 判断是否是命名空间列表请求
func isNamespaceListRequest(method, path string) bool {
	if method != "GET" {
		return false
	}
	// K8s Dashboard API 命名空间列表路径
	// Dashboard v2: /api/v1/namespace
	// K8s API: /api/v1/namespaces
	return path == "/api/v1/namespace" ||
		path == "/api/v1/namespaces" ||
		strings.HasPrefix(path, "/api/v1/namespace?") ||
		strings.HasPrefix(path, "/api/v1/namespaces?")
}

// namespaceFilterResponse 创建一个过滤命名空间列表响应的函数
func (h *ProxyHandler) namespaceFilterResponse(allowedNS []string) func(*http.Response) error {
	return func(resp *http.Response) error {
		if resp.StatusCode != http.StatusOK {
			return nil
		}

		var reader io.ReadCloser
		switch resp.Header.Get("Content-Encoding") {
		case "gzip":
			var err error
			reader, err = gzip.NewReader(resp.Body)
			if err != nil {
				return nil
			}
			defer reader.Close()
		default:
			reader = resp.Body
		}

		body, err := io.ReadAll(reader)
		resp.Body.Close()
		if err != nil {
			return nil
		}

		// 尝试解析为 Dashboard 命名空间列表格式
		filtered := filterNamespaceResponse(body, allowedNS)

		// 移除 gzip 编码（我们返回未压缩数据）
		resp.Header.Del("Content-Encoding")
		resp.Body = io.NopCloser(bytes.NewReader(filtered))
		resp.ContentLength = int64(len(filtered))
		resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(filtered)))
		return nil
	}
}

// filterNamespaceResponse 过滤命名空间列表响应
func filterNamespaceResponse(body []byte, allowedNS []string) []byte {
	allowedSet := map[string]bool{}
	for _, ns := range allowedNS {
		allowedSet[ns] = true
	}

	// 尝试 Dashboard v2 格式: {"namespaces": [{"objectMeta": {"name": "xxx"}, ...}]}
	var dashResp map[string]interface{}
	if err := json.Unmarshal(body, &dashResp); err != nil {
		return body
	}

	// Dashboard 格式: {"namespaces": [...]}
	if nsList, ok := dashResp["namespaces"]; ok {
		if nsArr, ok := nsList.([]interface{}); ok {
			filtered := make([]interface{}, 0)
			for _, item := range nsArr {
				if nsObj, ok := item.(map[string]interface{}); ok {
					nsName := extractNSName(nsObj)
					if nsName != "" && allowedSet[nsName] {
						filtered = append(filtered, item)
					}
				}
			}
			dashResp["namespaces"] = filtered
			// 更新 listMeta 中的 totalItems
			if listMeta, ok := dashResp["listMeta"].(map[string]interface{}); ok {
				listMeta["totalItems"] = len(filtered)
			}
			result, err := json.Marshal(dashResp)
			if err != nil {
				return body
			}
			return result
		}
	}

	// K8s 原生格式: {"kind": "NamespaceList", "items": [...]}
	if kind, ok := dashResp["kind"].(string); ok && kind == "NamespaceList" {
		if items, ok := dashResp["items"].([]interface{}); ok {
			filtered := make([]interface{}, 0)
			for _, item := range items {
				if nsObj, ok := item.(map[string]interface{}); ok {
					if meta, ok := nsObj["metadata"].(map[string]interface{}); ok {
						if name, ok := meta["name"].(string); ok && allowedSet[name] {
							filtered = append(filtered, item)
						}
					}
				}
			}
			dashResp["items"] = filtered
			result, err := json.Marshal(dashResp)
			if err != nil {
				return body
			}
			return result
		}
	}

	return body
}

// extractNSName 从命名空间对象中提取名称
func extractNSName(nsObj map[string]interface{}) string {
	// Dashboard 格式: {"objectMeta": {"name": "xxx"}}
	if objMeta, ok := nsObj["objectMeta"].(map[string]interface{}); ok {
		if name, ok := objMeta["name"].(string); ok {
			return name
		}
	}
	// 简单格式: {"name": "xxx"}
	if name, ok := nsObj["name"].(string); ok {
		return name
	}
	// K8s 格式: {"metadata": {"name": "xxx"}}
	if meta, ok := nsObj["metadata"].(map[string]interface{}); ok {
		if name, ok := meta["name"].(string); ok {
			return name
		}
	}
	return ""
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

// hasAnyWritePermission 检查 developer 是否拥有任意项目的写权限
func (h *ProxyHandler) hasAnyWritePermission(userID uint) bool {
	var count int64
	model.DB.Model(&model.UserProject{}).
		Where("user_id = ? AND permission = 'readwrite'", userID).
		Count(&count)
	return count > 0
}

func extractNamespace(path string) string {
	parts := splitPath(path)

	// K8s 原生 API 格式: .../namespaces/<ns>/...
	for i, p := range parts {
		if p == "namespaces" && i+1 < len(parts) {
			return parts[i+1]
		}
	}

	// Dashboard v2 自有 API 格式: /dashboard/api/v1/<resource>/<namespace>/<name>
	// 删除/编辑等操作走此路径，不含 "namespaces" 关键字
	dashboardResources := map[string]bool{
		"pod": true, "deployment": true, "service": true, "statefulset": true,
		"daemonset": true, "job": true, "cronjob": true, "replicaset": true,
		"replicationcontroller": true, "ingress": true, "configmap": true,
		"secret": true, "persistentvolumeclaim": true, "networkpolicy": true,
		"resourcequota": true, "limitrange": true, "serviceaccount": true,
		"role": true, "rolebinding": true, "horizontalpodautoscaler": true,
		"event": true, "endpoint": true,
	}
	for i, p := range parts {
		if p == "api" && i+2 < len(parts) && strings.HasPrefix(parts[i+1], "v") {
			if i+3 < len(parts) && dashboardResources[parts[i+2]] {
				return parts[i+3]
			}
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
