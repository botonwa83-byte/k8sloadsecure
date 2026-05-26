package middleware

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"k8sgate/model"
)

// getRealClientIP 从请求中获取真实客户端IP
func getRealClientIP(c *gin.Context) string {
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		ip := strings.TrimSpace(parts[0])
		if ip != "" {
			return ip
		}
	}
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return xri
	}
	return c.ClientIP()
}

var auditChan = make(chan model.AuditLog, 1000)

// auditCache 用于去重，避免短时间内重复记录相同操作
var auditCache = sync.Map{}

func init() {
	go auditWorker()
}

func auditWorker() {
	batch := make([]model.AuditLog, 0, 100)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case log := <-auditChan:
			batch = append(batch, log)
			if len(batch) >= 100 {
				flushAuditLogs(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				flushAuditLogs(batch)
				batch = batch[:0]
			}
		}
	}
}

func flushAuditLogs(logs []model.AuditLog) {
	if model.DB == nil || len(logs) == 0 {
		return
	}
	model.DB.Create(&logs)
}

// shouldLog 检查是否应该记录该操作（去重）
func shouldLog(userID uint, method, path string) bool {
	// 生成缓存键
	key := fmt.Sprintf("%d:%s:%s", userID, method, path)
	
	// 检查是否已存在
	if _, exists := auditCache.LoadOrStore(key, struct{}{}); exists {
		return false
	}
	
	// 30秒后自动删除缓存
	go func() {
		time.Sleep(30 * time.Second)
		auditCache.Delete(key)
	}()
	
	return true
}

// k8sPathRegex 解析标准 K8s API 路径
// /api/v1/namespaces/{ns}/pods/{name}
// /apis/apps/v1/namespaces/{ns}/deployments/{name}
var k8sPathRegex = regexp.MustCompile(
	`/(?:api|apis/[^/]+)/v[^/]+/(?:namespaces/([^/]+)/)?([^/]+)(?:/([^/]+))?`,
)

// dashboardPathRegex 解析 Dashboard 代理路径
// /dashboard/api/v1/_raw/deployment/namespace/{ns}/name/{name}
var dashboardPathRegex = regexp.MustCompile(
	`/dashboard/api/v1/_raw/([^/]+)/namespace/([^/]+)/name/([^/]+)`,
)

func parseK8sPath(path string) (namespace, resourceType, resourceName string) {
	// 先尝试匹配 Dashboard 路径格式
	dashboardMatches := dashboardPathRegex.FindStringSubmatch(path)
	if len(dashboardMatches) >= 4 {
		resourceType = singularize(dashboardMatches[1])
		namespace = dashboardMatches[2]
		resourceName = dashboardMatches[3]
		return
	}
	
	// 再尝试匹配标准 K8s API 路径格式
	matches := k8sPathRegex.FindStringSubmatch(path)
	if len(matches) >= 3 {
		namespace = matches[1]
		resourceType = singularize(matches[2])
		if len(matches) >= 4 {
			resourceName = matches[3]
		}
	}
	return
}

func singularize(resource string) string {
	mapping := map[string]string{
		"pods":                   "Pod",
		"services":              "Service",
		"deployments":           "Deployment",
		"replicasets":           "ReplicaSet",
		"statefulsets":          "StatefulSet",
		"daemonsets":            "DaemonSet",
		"jobs":                  "Job",
		"cronjobs":             "CronJob",
		"configmaps":           "ConfigMap",
		"secrets":              "Secret",
		"ingresses":            "Ingress",
		"namespaces":           "Namespace",
		"nodes":                "Node",
		"persistentvolumeclaims": "PersistentVolumeClaim",
		"endpoints":            "Endpoints",
		"events":               "Event",
	}
	if v, ok := mapping[strings.ToLower(resource)]; ok {
		return v
	}
	return resource
}

func actionDetail(method, resourceType, resourceName string) string {
	actionMap := map[string]string{
		"POST":   "创建",
		"PUT":    "更新",
		"PATCH":  "更新",
		"DELETE": "删除",
		"GET":    "查看",
	}
	verb := actionMap[method]
	if verb == "" {
		verb = method
	}
	if resourceName != "" {
		return verb + " " + resourceType + "/" + resourceName
	}
	if method == "GET" {
		return "查看 " + resourceType + " 列表"
	}
	return verb + " " + resourceType
}

func AuditLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		
		// 跳过非 dashboard 代理的请求和健康检查
		if !strings.HasPrefix(path, "/dashboard/") && !strings.HasPrefix(path, "/api/v1/proxy/") {
			c.Next()
			return
		}

		c.Next()

		userID := GetUserID(c)
		username := GetUsername(c)
		method := c.Request.Method

		// 优化1：跳过 GET/HEAD 请求（只读操作不记录）
		if method == "GET" || method == "HEAD" {
			return
		}

		// 优化2：去重检查，避免短时间内重复记录相同操作
		if !shouldLog(userID, method, path) {
			return
		}

		namespace, resourceType, resourceName := parseK8sPath(path)
		
		// 优化3：跳过无法识别的资源类型
		if resourceType == "" && resourceName == "" && namespace == "" {
			return
		}
		
		detail := actionDetail(method, resourceType, resourceName)

		auditChan <- model.AuditLog{
			UserID:       userID,
			Username:     username,
			Action:       method,
			ResourceType: resourceType,
			ResourceName: resourceName,
			Namespace:    namespace,
			RequestPath:  path,
			StatusCode:   c.Writer.Status(),
			ClientIP:     getRealClientIP(c),
			Detail:       detail,
			CreatedAt:    time.Now(),
		}
	}
}
