package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"k8sgate/config"
	"k8sgate/pkg"
	"k8sgate/service"
)

type UserHandler struct {
	cfg *config.Config
}

func NewUserHandler(cfg *config.Config) *UserHandler {
	return &UserHandler{cfg: cfg}
}

func (h *UserHandler) List(c *gin.Context) {
	var q service.UserListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}

	total, users, err := service.GetUserList(&q)
	if err != nil {
		pkg.Fail(c, http.StatusInternalServerError, 50001, "查询失败")
		return
	}
	pkg.OK(c, pkg.PageData(total, users))
}

func (h *UserHandler) Create(c *gin.Context) {
	var req service.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误: "+err.Error())
		return
	}

	user, err := service.CreateUser(&req, h.cfg.PasswordMaxAge)
	if err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	pkg.OK(c, user)
}

func (h *UserHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "无效的用户ID")
		return
	}

	var req service.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}

	if err := service.UpdateUser(uint(id), &req); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	pkg.OKMsg(c, "更新成功")
}

func (h *UserHandler) ResetPassword(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "无效的用户ID")
		return
	}

	var req struct {
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}

	if err := service.ResetPassword(uint(id), req.NewPassword, h.cfg.PasswordMaxAge); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	pkg.OKMsg(c, "密码重置成功")
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "无效的用户ID")
		return
	}

	if err := service.DeleteUser(uint(id)); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	pkg.OKMsg(c, "删除成功")
}
