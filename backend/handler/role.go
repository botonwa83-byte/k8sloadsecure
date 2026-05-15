package handler

import (
	"net/http"
	"strconv"

	"k8sgate/pkg"
	"k8sgate/service"

	"github.com/gin-gonic/gin"
)

type RoleHandler struct{}

// CreateRole 创建角色
func (h *RoleHandler) CreateRole(c *gin.Context) {
	var req service.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}

	if err := service.CreateRole(&req); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, err.Error())
		return
	}

	pkg.OK(c, nil)
}

// GetRole 获取角色详情
func (h *RoleHandler) GetRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}

	role, err := service.GetRole(uint(id))
	if err != nil {
		pkg.Fail(c, http.StatusNotFound, 40401, err.Error())
		return
	}

	pkg.OK(c, role)
}

// ListRoles 获取角色列表
func (h *RoleHandler) ListRoles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	total, roles, err := service.ListRoles(page, pageSize)
	if err != nil {
		pkg.Fail(c, http.StatusInternalServerError, 50001, err.Error())
		return
	}

	pkg.OK(c, gin.H{
		"total": total,
		"list":  roles,
	})
}

// UpdateRole 更新角色
func (h *RoleHandler) UpdateRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}

	var req service.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}

	if err := service.UpdateRole(uint(id), &req); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, err.Error())
		return
	}

	pkg.OK(c, nil)
}

// DeleteRole 删除角色
func (h *RoleHandler) DeleteRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}

	if err := service.DeleteRole(uint(id)); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, err.Error())
		return
	}

	pkg.OK(c, nil)
}

// AssignRole 分配角色给用户
func (h *RoleHandler) AssignRole(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}

	var req service.AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}

	if err := service.AssignRole(uint(userID), &req); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, err.Error())
		return
	}

	pkg.OK(c, nil)
}

// RemoveRole 移除用户角色
func (h *RoleHandler) RemoveRole(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}

	roleID, err := strconv.ParseUint(c.Param("role_id"), 10, 64)
	if err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}

	if err := service.RemoveRole(uint(userID), uint(roleID)); err != nil {
		pkg.Fail(c, http.StatusInternalServerError, 50001, err.Error())
		return
	}

	pkg.OK(c, nil)
}

// GetUserRoles 获取用户角色列表
func (h *RoleHandler) GetUserRoles(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}

	roles, err := service.GetUserRoles(uint(userID))
	if err != nil {
		pkg.Fail(c, http.StatusInternalServerError, 50001, err.Error())
		return
	}

	pkg.OK(c, roles)
}