package service

import (
	"errors"
	"time"

	"k8sgate/model"
	"k8sgate/pkg"
)

const (
	MaxFailedAttempts = 5
	LockDuration     = 30 * time.Minute
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

func Login(req *LoginRequest, clientIP string, passwordMaxAge int) (*model.User, string, error) {
	var user model.User
	result := model.DB.Where("username = ?", req.Username).First(&user)
	if result.Error != nil {
		recordLoginLog(0, req.Username, clientIP, "failed", "用户不存在")
		return nil, "", errors.New("用户名或密码错误")
	}

	if user.Status == "disabled" {
		recordLoginLog(user.ID, user.Username, clientIP, "failed", "账号已禁用")
		return nil, "", errors.New("账号已禁用")
	}

	if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
		recordLoginLog(user.ID, user.Username, clientIP, "locked", "账号锁定中")
		return nil, "", errors.New("账号已锁定，请30分钟后再试")
	}

	if !pkg.CheckPassword(req.Password, user.PasswordHash) {
		user.FailedLoginCount++
		updates := map[string]interface{}{"failed_login_count": user.FailedLoginCount}
		if user.FailedLoginCount >= MaxFailedAttempts {
			lockUntil := time.Now().Add(LockDuration)
			updates["locked_until"] = lockUntil
		}
		model.DB.Model(&user).Updates(updates)
		recordLoginLog(user.ID, user.Username, clientIP, "failed", "密码错误")
		return nil, "", errors.New("用户名或密码错误")
	}

	// 登录成功，重置失败计数
	model.DB.Model(&user).Updates(map[string]interface{}{
		"failed_login_count": 0,
		"locked_until":       nil,
	})

	token, err := pkg.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, "", errors.New("生成Token失败")
	}

	recordLoginLog(user.ID, user.Username, clientIP, "success", "")
	return &user, token, nil
}

func ChangePassword(userID uint, req *ChangePasswordRequest, passwordMaxAge int) error {
	var user model.User
	if err := model.DB.First(&user, userID).Error; err != nil {
		return errors.New("用户不存在")
	}

	if !pkg.CheckPassword(req.OldPassword, user.PasswordHash) {
		return errors.New("原密码错误")
	}

	if err := pkg.ValidatePassword(req.NewPassword); err != nil {
		return err
	}

	if pkg.CheckPassword(req.NewPassword, user.PasswordHash) {
		return errors.New("新密码不能与原密码相同")
	}

	hash, err := pkg.HashPassword(req.NewPassword)
	if err != nil {
		return errors.New("密码加密失败")
	}

	now := time.Now()
	expiresAt := now.AddDate(0, 0, passwordMaxAge)
	return model.DB.Model(&user).Updates(map[string]interface{}{
		"password_hash":       hash,
		"password_changed_at": now,
		"password_expires_at": expiresAt,
	}).Error
}

func IsPasswordExpired(user *model.User) bool {
	if user.PasswordExpiresAt == nil {
		return false
	}
	return time.Now().After(*user.PasswordExpiresAt)
}

func recordLoginLog(userID uint, username, clientIP, result, reason string) {
	model.DB.Create(&model.LoginLog{
		UserID:    userID,
		Username:  username,
		ClientIP:  clientIP,
		Result:    result,
		Reason:    reason,
		CreatedAt: time.Now(),
	})
}
