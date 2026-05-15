package middleware

import (
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"k8sgate/model"
)

var auditChan = make(chan model.AuditLog, 1000)

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

// k8sPathRegex 解析 K8s API 路径
// /api/v1/namespaces/{ns}/pods/{name}
// /apis/apps/v1/namespaces/{ns}/deployments/{name}
var k8sPathRegex = regexp.MustCompile(
	`/(?:api|apis/[^/]+)/v[^/]+/(?:namespaces/([^/]+)/)?([^/]+)(?:/([^/]+))?`,
)

func parseK8sPath(path string) (namespace, resourceType, resourceName string) {
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
		// 跳过非 dashboard 代理的请求和健康检查
		path := c.Request.URL.Path
		if !strings.HasPrefix(path, "/dashboard/") && !strings.HasPrefix(path, "/api/v1/proxy/") {
			c.Next()
			return
		}

		c.Next()

		userID := GetUserID(c)
		username := GetUsername(c)
		method := c.Request.Method

		namespace, resourceType, resourceName := parseK8sPath(path)
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
			ClientIP:     c.ClientIP(),
			Detail:       detail,
			CreatedAt:    time.Now(),
		}
	}
}
