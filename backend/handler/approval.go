package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"k8sgate/middleware"
	"k8sgate/pkg"
	"k8sgate/service"
)

type ApprovalHandler struct{}

func NewApprovalHandler() *ApprovalHandler {
	return &ApprovalHandler{}
}

// Submit 开发者提交写权限申请
func (h *ApprovalHandler) Submit(c *gin.Context) {
	var req service.SubmitRequestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误: "+err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	username := middleware.GetUsername(c)

	if err := service.SubmitPermissionRequest(userID, username, &req); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	pkg.OKMsg(c, "申请已提交，等待管理员审批")
}

// MyRequests 查看我的申请
func (h *ApprovalHandler) MyRequests(c *gin.Context) {
	var q service.RequestListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}
	q.UserID = middleware.GetUserID(c)

	total, requests, err := service.GetRequestList(&q)
	if err != nil {
		pkg.Fail(c, http.StatusInternalServerError, 50001, "查询失败")
		return
	}
	pkg.OK(c, pkg.PageData(total, requests))
}

// List 管理员查看所有申请
func (h *ApprovalHandler) List(c *gin.Context) {
	var q service.RequestListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}

	total, requests, err := service.GetRequestList(&q)
	if err != nil {
		pkg.Fail(c, http.StatusInternalServerError, 50001, "查询失败")
		return
	}
	pkg.OK(c, pkg.PageData(total, requests))
}

// Review 管理员审批
func (h *ApprovalHandler) Review(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "无效的申请ID")
		return
	}

	var req service.ReviewRequestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误: "+err.Error())
		return
	}

	reviewerID := middleware.GetUserID(c)
	reviewerName := middleware.GetUsername(c)

	if err := service.ReviewPermissionRequest(uint(id), reviewerID, reviewerName, &req); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	pkg.OKMsg(c, "审批完成")
}
