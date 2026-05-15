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
	ensureSystemRoles()

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
	roleH := &handler.RoleHandler{}

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

				// 角色管理
				admin.GET("/roles", roleH.ListRoles)
				admin.POST("/roles", roleH.CreateRole)
				admin.GET("/roles/:id", roleH.GetRole)
				admin.PUT("/roles/:id", roleH.UpdateRole)
				admin.DELETE("/roles/:id", roleH.DeleteRole)

				// 用户角色管理
				admin.GET("/user-roles/:user_id", roleH.GetUserRoles)
				admin.POST("/user-roles/:user_id", roleH.AssignRole)
				admin.DELETE("/user-roles/:user_id/:role_id", roleH.RemoveRole)
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

func ensureSystemRoles() {
	var count int64
	model.DB.Model(&model.Role{}).Where("type = ?", "system").Count(&count)
	if count > 0 {
		return
	}

	roles := []model.Role{
		{Name: "super_admin", Description: "超级管理员", Type: "system"},
		{Name: "admin", Description: "管理员", Type: "system", ParentID: 1},
		{Name: "security_admin", Description: "安全管理员", Type: "system", ParentID: 2},
		{Name: "ops_admin", Description: "运维管理员", Type: "system", ParentID: 2},
		{Name: "global_viewer", Description: "全局只读", Type: "system"},
		{Name: "project_admin", Description: "项目管理员", Type: "system", ParentID: 2},
		{Name: "project_developer", Description: "项目开发者", Type: "system", ParentID: 6},
		{Name: "project_viewer", Description: "项目查看者", Type: "system", ParentID: 6},
		{Name: "project_operator", Description: "项目运维", Type: "system", ParentID: 6},
	}

	for _, role := range roles {
		if err := model.DB.Create(&role).Error; err != nil {
			log.Printf("Warning: failed to create role %s: %v", role.Name, err)
		}
	}

	permissions := []model.RolePermission{
		// super_admin: 所有权限
		{RoleID: 1, Resource: "*", Actions: "[\"*\"]"},
		// admin: 大部分权限
		{RoleID: 2, Resource: "user", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		{RoleID: 2, Resource: "role", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		{RoleID: 2, Resource: "project", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		{RoleID: 2, Resource: "namespace", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		{RoleID: 2, Resource: "pod", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		{RoleID: 2, Resource: "deployment", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		{RoleID: 2, Resource: "service", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		{RoleID: 2, Resource: "configmap", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		{RoleID: 2, Resource: "secret", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		{RoleID: 2, Resource: "audit", Actions: "[\"view\",\"create\",\"update\",\"delete\",\"export\"]"},
		{RoleID: 2, Resource: "approval", Actions: "[\"view\",\"approve\"]"},
		// security_admin: 审计和权限管理
		{RoleID: 3, Resource: "user", Actions: "[\"view\"]"},
		{RoleID: 3, Resource: "role", Actions: "[\"view\"]"},
		{RoleID: 3, Resource: "audit", Actions: "[\"view\",\"export\"]"},
		{RoleID: 3, Resource: "approval", Actions: "[\"view\",\"approve\"]"},
		// ops_admin: K8s资源管理
		{RoleID: 4, Resource: "namespace", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		{RoleID: 4, Resource: "pod", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		{RoleID: 4, Resource: "deployment", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		{RoleID: 4, Resource: "service", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		{RoleID: 4, Resource: "configmap", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		{RoleID: 4, Resource: "secret", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		{RoleID: 4, Resource: "audit", Actions: "[\"view\"]"},
		// global_viewer: 全局只读
		{RoleID: 5, Resource: "user", Actions: "[\"view\"]"},
		{RoleID: 5, Resource: "role", Actions: "[\"view\"]"},
		{RoleID: 5, Resource: "project", Actions: "[\"view\"]"},
		{RoleID: 5, Resource: "namespace", Actions: "[\"view\"]"},
		{RoleID: 5, Resource: "pod", Actions: "[\"view\"]"},
		{RoleID: 5, Resource: "deployment", Actions: "[\"view\"]"},
		{RoleID: 5, Resource: "service", Actions: "[\"view\"]"},
		{RoleID: 5, Resource: "configmap", Actions: "[\"view\"]"},
		{RoleID: 5, Resource: "secret", Actions: "[\"view\"]"},
		{RoleID: 5, Resource: "audit", Actions: "[\"view\",\"export\"]"},
		// project_admin: 项目内全部权限
		{RoleID: 6, Resource: "project", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		{RoleID: 6, Resource: "namespace", Actions: "[\"view\"]"},
		{RoleID: 6, Resource: "pod", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		{RoleID: 6, Resource: "deployment", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		{RoleID: 6, Resource: "service", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		{RoleID: 6, Resource: "configmap", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		{RoleID: 6, Resource: "secret", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		// project_developer: 开发相关权限
		{RoleID: 7, Resource: "project", Actions: "[\"view\"]"},
		{RoleID: 7, Resource: "namespace", Actions: "[\"view\"]"},
		{RoleID: 7, Resource: "pod", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		{RoleID: 7, Resource: "deployment", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		{RoleID: 7, Resource: "service", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		{RoleID: 7, Resource: "configmap", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		{RoleID: 7, Resource: "secret", Actions: "[\"view\"]"},
		{RoleID: 7, Resource: "approval", Actions: "[\"view\"]"},
		// project_viewer: 只读
		{RoleID: 8, Resource: "project", Actions: "[\"view\"]"},
		{RoleID: 8, Resource: "namespace", Actions: "[\"view\"]"},
		{RoleID: 8, Resource: "pod", Actions: "[\"view\"]"},
		{RoleID: 8, Resource: "deployment", Actions: "[\"view\"]"},
		{RoleID: 8, Resource: "service", Actions: "[\"view\"]"},
		{RoleID: 8, Resource: "configmap", Actions: "[\"view\"]"},
		{RoleID: 8, Resource: "secret", Actions: "[\"view\"]"},
		// project_operator: 运维相关权限
		{RoleID: 9, Resource: "project", Actions: "[\"view\"]"},
		{RoleID: 9, Resource: "namespace", Actions: "[\"view\"]"},
		{RoleID: 9, Resource: "pod", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		{RoleID: 9, Resource: "deployment", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		{RoleID: 9, Resource: "service", Actions: "[\"view\",\"create\",\"update\",\"delete\"]"},
		{RoleID: 9, Resource: "configmap", Actions: "[\"view\"]"},
		{RoleID: 9, Resource: "secret", Actions: "[\"view\"]"},
	}

	for _, perm := range permissions {
		if err := model.DB.Create(&perm).Error; err != nil {
			log.Printf("Warning: failed to create permission for role %d: %v", perm.RoleID, err)
		}
	}

	log.Println("System roles created successfully")
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
