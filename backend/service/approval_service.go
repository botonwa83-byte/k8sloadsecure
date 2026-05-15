package service

import (
	"errors"
	"time"

	"k8sgate/model"
)

type SubmitRequestReq struct {
	ProjectID uint   `json:"project_id" binding:"required"`
	Reason    string `json:"reason" binding:"required"`
}

type ReviewRequestReq struct {
	Status     string `json:"status" binding:"required,oneof=approved rejected"`
	ReviewNote string `json:"review_note"`
	ExpireDays int    `json:"expire_days"` // 审批通过后写权限有效天数，0表示不限
}

type RequestListQuery struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
	Status   string `form:"status"`
	UserID   uint   `form:"user_id"`
}

func SubmitPermissionRequest(userID uint, username string, req *SubmitRequestReq) error {
	// 检查项目是否存在
	var project model.Project
	if err := model.DB.First(&project, req.ProjectID).Error; err != nil {
		return errors.New("项目不存在")
	}

	// 检查用户是否已分配到该项目
	var up model.UserProject
	result := model.DB.Where("user_id = ? AND project_id = ?", userID, req.ProjectID).First(&up)
	if result.Error != nil {
		return errors.New("你未被分配到该项目")
	}

	// 检查是否已有读写权限
	if up.Permission == "readwrite" {
		return errors.New("你已拥有该项目的写权限")
	}

	// 检查是否已有待审批的申请
	var pendingCount int64
	model.DB.Model(&model.PermissionRequest{}).
		Where("user_id = ? AND project_id = ? AND status = 'pending'", userID, req.ProjectID).
		Count(&pendingCount)
	if pendingCount > 0 {
		return errors.New("已有待审批的申请，请勿重复提交")
	}

	return model.DB.Create(&model.PermissionRequest{
		UserID:    userID,
		Username:  username,
		ProjectID: req.ProjectID,
		Reason:    req.Reason,
		Status:    "pending",
	}).Error
}

func ReviewPermissionRequest(requestID uint, reviewerID uint, reviewerName string, req *ReviewRequestReq) error {
	var pr model.PermissionRequest
	if err := model.DB.First(&pr, requestID).Error; err != nil {
		return errors.New("申请不存在")
	}
	if pr.Status != "pending" {
		return errors.New("该申请已处理")
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":      req.Status,
		"reviewer_id": reviewerID,
		"reviewer":    reviewerName,
		"review_note": req.ReviewNote,
		"reviewed_at": now,
	}

	if req.Status == "approved" {
		// 设置过期时间
		if req.ExpireDays > 0 {
			expiresAt := now.AddDate(0, 0, req.ExpireDays)
			updates["expires_at"] = expiresAt
		}

		// 更新 user_projects 权限为 readwrite
		model.DB.Model(&model.UserProject{}).
			Where("user_id = ? AND project_id = ?", pr.UserID, pr.ProjectID).
			Update("permission", "readwrite")
	}

	return model.DB.Model(&pr).Updates(updates).Error
}

func GetRequestList(q *RequestListQuery) (int64, []model.PermissionRequest, error) {
	var total int64
	var requests []model.PermissionRequest

	db := model.DB.Model(&model.PermissionRequest{})
	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}
	if q.UserID > 0 {
		db = db.Where("user_id = ?", q.UserID)
	}

	db.Count(&total)

	offset := (q.Page - 1) * q.PageSize
	if offset < 0 {
		offset = 0
	}
	err := db.Order("id DESC").Offset(offset).Limit(q.PageSize).
		Preload("Project").Find(&requests).Error
	return total, requests, err
}

// ExpireWritePermissions 检查并回收过期的写权限
func ExpireWritePermissions() int64 {
	now := time.Now()

	// 找到已过期的审批记录
	var expired []model.PermissionRequest
	model.DB.Where("status = 'approved' AND expires_at IS NOT NULL AND expires_at < ?", now).
		Find(&expired)

	var count int64
	for _, pr := range expired {
		// 回收写权限，改回只读
		model.DB.Model(&model.UserProject{}).
			Where("user_id = ? AND project_id = ?", pr.UserID, pr.ProjectID).
			Update("permission", "read")

		// 标记审批记录为已过期
		model.DB.Model(&pr).Update("status", "expired")
		count++
	}
	return count
}
