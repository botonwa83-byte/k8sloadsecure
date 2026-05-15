package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"k8sgate/middleware"
	"k8sgate/pkg"
	"k8sgate/service"
)

type AuditHandler struct{}

func NewAuditHandler() *AuditHandler {
	return &AuditHandler{}
}

func (h *AuditHandler) Logs(c *gin.Context) {
	var q service.AuditLogQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}

	// 非管理员只能看自己的日志
	if middleware.GetRole(c) != "admin" {
		q.UserID = middleware.GetUserID(c)
	}

	total, logs, err := service.GetAuditLogs(&q)
	if err != nil {
		pkg.Fail(c, http.StatusInternalServerError, 50001, "查询失败")
		return
	}
	pkg.OK(c, pkg.PageData(total, logs))
}

func (h *AuditHandler) Report(c *gin.Context) {
	var q service.ReportQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误: 需要 start_time 和 end_time")
		return
	}

	// 非管理员只能看自己的报告
	if middleware.GetRole(c) != "admin" {
		q.UserID = middleware.GetUserID(c)
	}
	if q.UserID == 0 {
		pkg.Fail(c, http.StatusBadRequest, 40001, "请指定 user_id")
		return
	}

	report, err := service.GetUserReport(&q)
	if err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	pkg.OK(c, report)
}

func (h *AuditHandler) Export(c *gin.Context) {
	var q service.AuditLogQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}

	if middleware.GetRole(c) != "admin" {
		q.UserID = middleware.GetUserID(c)
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=audit_logs.csv")
	// 写入 BOM 以便 Excel 正确识别 UTF-8
	c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})
	service.ExportAuditCSV(c.Writer, &q)
}

func (h *AuditHandler) Stats(c *gin.Context) {
	var q service.StatsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误: 需要 start_time 和 end_time")
		return
	}

	stats, err := service.GetGlobalStats(&q)
	if err != nil {
		pkg.Fail(c, http.StatusInternalServerError, 50001, "查询失败")
		return
	}
	pkg.OK(c, stats)
}

func (h *AuditHandler) LoginLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	username := c.Query("username")
	result := c.Query("result")
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")

	total, logs, err := service.GetLoginLogs(page, pageSize, username, result, startTime, endTime)
	if err != nil {
		pkg.Fail(c, http.StatusInternalServerError, 50001, "查询失败")
		return
	}
	pkg.OK(c, pkg.PageData(total, logs))
}
