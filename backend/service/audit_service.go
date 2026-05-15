package service

import (
	"encoding/csv"
	"fmt"
	"io"
	"time"

	"k8sgate/model"

	"gorm.io/gorm"
)

type AuditLogQuery struct {
	Page      int    `form:"page,default=1"`
	PageSize  int    `form:"page_size,default=50"`
	UserID    uint   `form:"user_id"`
	Action    string `form:"action"`
	Namespace string `form:"namespace"`
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
}

type ReportQuery struct {
	UserID    uint   `form:"user_id"`
	StartTime string `form:"start_time" binding:"required"`
	EndTime   string `form:"end_time" binding:"required"`
}

type StatsQuery struct {
	StartTime string `form:"start_time" binding:"required"`
	EndTime   string `form:"end_time" binding:"required"`
}

func GetAuditLogs(q *AuditLogQuery) (int64, []model.AuditLog, error) {
	var total int64
	var logs []model.AuditLog

	db := model.DB.Model(&model.AuditLog{})
	db = applyAuditFilters(db, q.UserID, q.Action, q.Namespace, q.StartTime, q.EndTime)

	db.Count(&total)

	offset := (q.Page - 1) * q.PageSize
	if offset < 0 {
		offset = 0
	}
	err := db.Order("id DESC").Offset(offset).Limit(q.PageSize).Find(&logs).Error
	return total, logs, err
}

func GetUserReport(q *ReportQuery) (map[string]interface{}, error) {
	var user model.User
	if err := model.DB.First(&user, q.UserID).Error; err != nil {
		return nil, fmt.Errorf("用户不存在")
	}

	db := model.DB.Model(&model.AuditLog{}).Where("user_id = ?", q.UserID)
	if q.StartTime != "" {
		db = db.Where("created_at >= ?", q.StartTime)
	}
	if q.EndTime != "" {
		db = db.Where("created_at <= ?", q.EndTime+" 23:59:59")
	}

	// 总操作数
	var totalOps int64
	db.Count(&totalOps)

	// 按操作类型统计
	type ActionCount struct {
		Action string `json:"action"`
		Cnt    int64  `json:"cnt"`
	}
	var actionCounts []ActionCount
	db.Select("action, COUNT(*) as cnt").Group("action").Find(&actionCounts)
	byAction := map[string]int64{}
	for _, ac := range actionCounts {
		byAction[ac.Action] = ac.Cnt
	}

	// 按结果统计
	var successCount, failedCount, deniedCount int64
	db.Where("status_code BETWEEN 200 AND 299").Count(&successCount)
	db.Where("status_code = 403").Count(&deniedCount)
	db.Where("status_code >= 400 AND status_code != 403").Count(&failedCount)

	// 活跃命名空间 TOP 5
	type NsCount struct {
		Namespace string `json:"namespace"`
		Cnt       int64  `json:"cnt"`
	}
	var nsCounts []NsCount
	db.Select("namespace, COUNT(*) as cnt").
		Where("namespace != ''").
		Group("namespace").Order("cnt DESC").Limit(5).Find(&nsCounts)

	activeNs := make([]string, len(nsCounts))
	for i, ns := range nsCounts {
		activeNs[i] = ns.Namespace
	}

	// 活跃天数
	var activeDays int64
	db.Select("COUNT(DISTINCT DATE(created_at))").Row().Scan(&activeDays)

	// 敏感操作
	var sensitiveOps []model.AuditLog
	sensitiveDB := model.DB.Model(&model.AuditLog{}).Where("user_id = ?", q.UserID)
	if q.StartTime != "" {
		sensitiveDB = sensitiveDB.Where("created_at >= ?", q.StartTime)
	}
	if q.EndTime != "" {
		sensitiveDB = sensitiveDB.Where("created_at <= ?", q.EndTime+" 23:59:59")
	}
	sensitiveDB.Where(
		"action = 'DELETE' OR status_code = 403 OR namespace IN ('kube-system','istio-system','monitoring')",
	).Order("created_at DESC").Limit(50).Find(&sensitiveOps)

	return map[string]interface{}{
		"user_id":      user.ID,
		"username":     user.Username,
		"display_name": user.DisplayName,
		"period": map[string]string{
			"start": q.StartTime,
			"end":   q.EndTime,
		},
		"summary": map[string]interface{}{
			"total_operations":  totalOps,
			"by_action":         byAction,
			"by_result": map[string]int64{
				"success": successCount,
				"failed":  failedCount,
				"denied":  deniedCount,
			},
			"active_namespaces": activeNs,
			"active_days":       activeDays,
		},
		"sensitive_operations": sensitiveOps,
	}, nil
}

func ExportAuditCSV(w io.Writer, q *AuditLogQuery) error {
	var logs []model.AuditLog

	db := model.DB.Model(&model.AuditLog{})
	db = applyAuditFilters(db, q.UserID, q.Action, q.Namespace, q.StartTime, q.EndTime)
	db.Order("created_at DESC").Limit(10000).Find(&logs)

	writer := csv.NewWriter(w)
	defer writer.Flush()

	writer.Write([]string{"时间", "用户名", "操作类型", "资源类型", "资源名", "命名空间", "状态码", "客户端IP", "操作摘要"})
	for _, log := range logs {
		writer.Write([]string{
			log.CreatedAt.Format(time.DateTime),
			log.Username,
			log.Action,
			log.ResourceType,
			log.ResourceName,
			log.Namespace,
			fmt.Sprintf("%d", log.StatusCode),
			log.ClientIP,
			log.Detail,
		})
	}
	return nil
}

func GetGlobalStats(q *StatsQuery) (map[string]interface{}, error) {
	db := model.DB.Model(&model.AuditLog{})
	if q.StartTime != "" {
		db = db.Where("created_at >= ?", q.StartTime)
	}
	if q.EndTime != "" {
		db = db.Where("created_at <= ?", q.EndTime+" 23:59:59")
	}

	var totalOps int64
	db.Count(&totalOps)

	var activeUsers int64
	db.Select("COUNT(DISTINCT user_id)").Row().Scan(&activeUsers)

	type UserCount struct {
		Username string `json:"username"`
		Count    int64  `json:"count"`
	}
	var topUsers []UserCount
	db.Select("username, COUNT(*) as count").Group("username").Order("count DESC").Limit(20).Find(&topUsers)

	type NsCount struct {
		Namespace string `json:"namespace"`
		Count     int64  `json:"count"`
	}
	var topNs []NsCount
	db.Select("namespace, COUNT(*) as count").Where("namespace != ''").Group("namespace").Order("count DESC").Limit(20).Find(&topNs)

	var deniedOps int64
	db.Where("status_code = 403").Count(&deniedOps)

	type ActionCount struct {
		Action string `json:"action"`
		Count  int64  `json:"count"`
	}
	var byAction []ActionCount
	db.Select("action, COUNT(*) as count").Group("action").Find(&byAction)
	actionMap := map[string]int64{}
	for _, a := range byAction {
		actionMap[a.Action] = a.Count
	}

	return map[string]interface{}{
		"total_operations":  totalOps,
		"active_users":      activeUsers,
		"top_users":         topUsers,
		"top_namespaces":    topNs,
		"denied_operations": deniedOps,
		"by_action":         actionMap,
	}, nil
}

func GetLoginLogs(page, pageSize int, username, result, startTime, endTime string) (int64, []model.LoginLog, error) {
	var total int64
	var logs []model.LoginLog

	db := model.DB.Model(&model.LoginLog{})
	if username != "" {
		db = db.Where("username = ?", username)
	}
	if result != "" {
		db = db.Where("result = ?", result)
	}
	if startTime != "" {
		db = db.Where("created_at >= ?", startTime)
	}
	if endTime != "" {
		db = db.Where("created_at <= ?", endTime+" 23:59:59")
	}

	db.Count(&total)
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	err := db.Order("id DESC").Offset(offset).Limit(pageSize).Find(&logs).Error
	return total, logs, err
}

func applyAuditFilters(db *gorm.DB, userID uint, action, namespace, startTime, endTime string) *gorm.DB {
	if userID > 0 {
		db = db.Where("user_id = ?", userID)
	}
	if action != "" {
		db = db.Where("action = ?", action)
	}
	if namespace != "" {
		db = db.Where("namespace = ?", namespace)
	}
	if startTime != "" {
		db = db.Where("created_at >= ?", startTime)
	}
	if endTime != "" {
		db = db.Where("created_at <= ?", endTime+" 23:59:59")
	}
	return db
}
