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
	"k8sgate/service"
)

func main() {
	cfg := config.Load()

	model.InitDB(cfg.DSN())
	pkg.InitJWT(cfg.JWTSecret)
	k8sclient.InitClient()

	if err := k8sclient.InitK8sResources(); err != nil {
		log.Printf("Warning: failed to init K8s resources: %v (will retry later)", err)
	}

	ensureDefaultAdmin(cfg.PasswordMaxAge)

	// 启动过期权限回收定时任务
	go expirePermissionTicker()

	r := gin.Default()

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

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
	approvalH := handler.NewApprovalHandler()

	v1 := r.Group("/api/v1")
	{
		v1.POST("/auth/login", authH.Login)

		auth := v1.Group("", middleware.AuthRequired())
		{
			auth.POST("/auth/logout", authH.Logout)
			auth.PUT("/auth/password", authH.ChangePassword)
			auth.GET("/auth/me", authH.Me)

			// 审计日志（developer只看自己的，global_viewer和admin看所有）
			auth.GET("/audit/logs", auditH.Logs)
			auth.GET("/audit/report", auditH.Report)
			auth.GET("/audit/export", auditH.Export)

			// 写权限申请（developer 使用）
			auth.POST("/approval/submit", approvalH.Submit)
			auth.GET("/approval/my-requests", approvalH.MyRequests)

			// 管理员接口
			admin := auth.Group("", middleware.AdminRequired())
			{
				admin.GET("/users", userH.List)
				admin.POST("/users", userH.Create)
				admin.PUT("/users/:id", userH.Update)
				admin.PUT("/users/:id/reset-password", userH.ResetPassword)
				admin.DELETE("/users/:id", userH.Delete)

				admin.GET("/projects", projectH.List)
				admin.POST("/projects", projectH.Create)
				admin.GET("/projects/:id", projectH.Get)
				admin.PUT("/projects/:id", projectH.Update)
				admin.DELETE("/projects/:id", projectH.Delete)
				admin.POST("/projects/:id/users", projectH.AssignUser)
				admin.DELETE("/projects/:id/users/:user_id", projectH.RemoveUser)

				admin.GET("/namespaces", projectH.ListNamespaces)

				admin.GET("/audit/stats", auditH.Stats)
				admin.GET("/login-logs", auditH.LoginLogs)

				// 审批管理
				admin.GET("/approval/requests", approvalH.List)
				admin.PUT("/approval/requests/:id", approvalH.Review)
			}
		}
	}

	dashboard := r.Group("/dashboard", middleware.AuthRequired(), middleware.AuditLog())
	{
		dashboard.Any("/*path", proxyH.Proxy)
	}

	// 公共统计接口（放在v1组内）
	v1.GET("/stats/dashboard", middleware.AuthRequired(), auditH.DashboardStats)
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

// expirePermissionTicker 每小时检查并回收过期的写权限
func expirePermissionTicker() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		count := service.ExpireWritePermissions()
		if count > 0 {
			log.Printf("Expired %d write permissions", count)
		}
	}
}
