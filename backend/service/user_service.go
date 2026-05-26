package service

import (
	"errors"
	"time"

	"k8sgate/model"
	"k8sgate/pkg"
)

type CreateUserRequest struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`
	Email       string `json:"email"`
	Remark      string `json:"remark"`
	IsAdmin     bool   `json:"is_admin"`
}

type UpdateUserRequest struct {
	DisplayName *string `json:"display_name"`
	Email       *string `json:"email"`
	Remark      *string `json:"remark"`
	IsAdmin     *bool   `json:"is_admin"`
	Status      *string `json:"status" binding:"omitempty,oneof=active disabled"`
}

type UserListQuery struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
	Keyword  string `form:"keyword"`
	Status   string `form:"status"`
}

func CreateUser(req *CreateUserRequest, passwordMaxAge int) (*model.User, error) {
	if err := pkg.ValidatePassword(req.Password); err != nil {
		return nil, err
	}

	// 检查是否存在同名的活跃用户
	var count int64
	model.DB.Model(&model.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		return nil, errors.New("用户名已存在")
	}
	// 清理同名的软删除记录，避免唯一索引冲突
	model.DB.Unscoped().Where("username = ? AND deleted_at IS NOT NULL", req.Username).Delete(&model.User{})

	hash, err := pkg.HashPassword(req.Password)
	if err != nil {
		return nil, errors.New("密码加密失败")
	}

	role := "developer"
	if req.IsAdmin {
		role = "admin"
	}

	now := time.Now()
	expiresAt := now.AddDate(0, 0, passwordMaxAge)
	user := model.User{
		Username:          req.Username,
		PasswordHash:      hash,
		DisplayName:       req.DisplayName,
		Email:             req.Email,
		Role:              role,
		Remark:            req.Remark,
		Status:            "active",
		PasswordChangedAt: &now,
		PasswordExpiresAt: &expiresAt,
	}

	if err := model.DB.Create(&user).Error; err != nil {
		return nil, errors.New("创建用户失败")
	}
	return &user, nil
}

func GetUserList(q *UserListQuery) (int64, []model.User, error) {
	var total int64
	var users []model.User

	db := model.DB.Model(&model.User{})
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		db = db.Where("username LIKE ? OR display_name LIKE ? OR email LIKE ?", like, like, like)
	}
	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}

	db.Count(&total)

	offset := (q.Page - 1) * q.PageSize
	if offset < 0 {
		offset = 0
	}
	err := db.Order("id DESC").Offset(offset).Limit(q.PageSize).Find(&users).Error
	return total, users, err
}

func UpdateUser(id uint, req *UpdateUserRequest) error {
	var user model.User
	if err := model.DB.First(&user, id).Error; err != nil {
		return errors.New("用户不存在")
	}

	updates := map[string]interface{}{}
	if req.DisplayName != nil {
		updates["display_name"] = *req.DisplayName
	}
	if req.Email != nil {
		updates["email"] = *req.Email
	}
	if req.Remark != nil {
		updates["remark"] = *req.Remark
	}
	if req.IsAdmin != nil {
		if *req.IsAdmin {
			updates["role"] = "admin"
		} else {
			updates["role"] = "developer"
		}
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if len(updates) == 0 {
		return nil
	}
	return model.DB.Model(&user).Updates(updates).Error
}

func ResetPassword(id uint, newPassword string, passwordMaxAge int) error {
	if err := pkg.ValidatePassword(newPassword); err != nil {
		return err
	}

	var user model.User
	if err := model.DB.First(&user, id).Error; err != nil {
		return errors.New("用户不存在")
	}

	hash, err := pkg.HashPassword(newPassword)
	if err != nil {
		return errors.New("密码加密失败")
	}

	now := time.Now()
	expiresAt := now.AddDate(0, 0, passwordMaxAge)
	return model.DB.Model(&user).Updates(map[string]interface{}{
		"password_hash":       hash,
		"password_changed_at": now,
		"password_expires_at": expiresAt,
		"failed_login_count":  0,
		"locked_until":        nil,
	}).Error
}

func DeleteUser(id uint) error {
	var user model.User
	if err := model.DB.First(&user, id).Error; err != nil {
		return errors.New("用户不存在")
	}
	if user.Role == "admin" {
		var count int64
		model.DB.Model(&model.User{}).Where("role = ? AND id != ?", "admin", id).Count(&count)
		if count == 0 {
			return errors.New("不能删除最后一个管理员")
		}
	}
	// 同时删除用户的项目分配
	model.DB.Where("user_id = ?", id).Delete(&model.UserProject{})
	return model.DB.Delete(&user).Error
}
