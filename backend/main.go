package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"k8sgate/config"
	"k8sgate/handler"
	k8sclient "k8sgate/k8s"
	"k8sgate/middleware"
	"k8sgate/model"
	"k8sgate/pkg"
)

func main() {
	cfg := config.Load()

	// 初始化数据库
	model.InitDB(cfg.DSN())

	// 初始化 JWT
	pkg.InitJWT(cfg.JWTSecret)

	// 初始化 K8s 客户端
	k8sclient.InitClient()

	// 初始化 K8s 资源（命名空间、ClusterRole、Admin SA）
	if err := k8sclient.InitK8sResources(); err != nil {
		log.Printf("Warning: failed to init K8s resources: %v (will retry later)", err)
	}

	// 创建默认管理员账号
	ensureDefaultAdmin(cfg.PasswordMaxAge)

	r := gin.Default()

	// 健康检查
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 注册路由
	registerRoutes(r, cfg)

	log.Printf("K8sGate starting on port %d", cfg.ServerPort)
	if err := r.Run(fmt.Sprintf(":%d", cfg.ServerPort)); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func registerRoutes(r *gin.Engine, cfg *config.Config) {
	authH := handler.NewAuthHandler(cfg)
	userH := handler.NewUserHandler(cfg)
	projectH := handler.NewProjectHandler()
	auditH := handler.NewAuditHandler()
	proxyH := handler.NewProxyHandler(cfg)

	v1 := r.Group("/api/v1")
	{
		// 公开接口
		v1.POST("/auth/login", authH.Login)

		// 需要登录的接口
		auth := v1.Group("", middleware.AuthRequired())
		{
			auth.POST("/auth/logout", authH.Logout)
			auth.PUT("/auth/password", authH.ChangePassword)
			auth.GET("/auth/me", authH.Me)

			// 审计日志（所有登录用户可访问，非管理员只能看自己的）
			auth.GET("/audit/logs", auditH.Logs)
			auth.GET("/audit/report", auditH.Report)
			auth.GET("/audit/export", auditH.Export)

			// 管理员接口
			admin := auth.Group("", middleware.AdminRequired())
			{
				// 用户管理
				admin.GET("/users", userH.List)
				admin.POST("/users", userH.Create)
				admin.PUT("/users/:id", userH.Update)
				admin.PUT("/users/:id/reset-password", userH.ResetPassword)
				admin.DELETE("/users/:id", userH.Delete)

				// 项目管理
				admin.GET("/projects", projectH.List)
				admin.POST("/projects", projectH.Create)
				admin.GET("/projects/:id", projectH.Get)
				admin.PUT("/projects/:id", projectH.Update)
				admin.DELETE("/projects/:id", projectH.Delete)
				admin.POST("/projects/:id/users", projectH.AssignUser)
				admin.DELETE("/projects/:id/users/:user_id", projectH.RemoveUser)

				// 命名空间列表（从 K8s 实时获取）
				admin.GET("/namespaces", projectH.ListNamespaces)

				// 全局统计和登录日志
				admin.GET("/audit/stats", auditH.Stats)
				admin.GET("/login-logs", auditH.LoginLogs)
			}
		}
	}

	// Dashboard 代理
	dashboard := r.Group("/dashboard", middleware.AuthRequired(), middleware.AuditLog())
	{
		dashboard.Any("/*path", proxyH.Proxy)
	}
}

func ensureDefaultAdmin(passwordMaxAge int) {
	var count int64
	model.DB.Model(&model.User{}).Where("role = ?", "admin").Count(&count)
	if count > 0 {
		return
	}

	hash, _ := pkg.HashPassword("Admin@123")
	now := time.Now()
	expiresAt := now.AddDate(0, 0, passwordMaxAge)
	admin := model.User{
		Username:          "admin",
		PasswordHash:      hash,
		DisplayName:       "系统管理员",
		Role:              "admin",
		Status:            "active",
		PasswordChangedAt: &now,
		PasswordExpiresAt: &expiresAt,
	}
	if err := model.DB.Create(&admin).Error; err != nil {
		log.Printf("Warning: failed to create default admin: %v", err)
	} else {
		log.Println("Default admin created: admin / Admin@123")
	}
}

