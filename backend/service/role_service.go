package service

import (
	"encoding/json"
	"errors"
	"k8sgate/model"
)

// CreateRoleRequest 创建角色请求
type CreateRoleRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	ParentID    uint     `json:"parent_id"`
	Permissions []string `json:"permissions"` // 格式: "resource:action1,action2"
}

// UpdateRoleRequest 更新角色请求
type UpdateRoleRequest struct {
	Name        *string   `json:"name"`
	Description *string   `json:"description"`
	ParentID    *uint     `json:"parent_id"`
	Permissions *[]string `json:"permissions"`
}

// AssignRoleRequest 分配角色请求
type AssignRoleRequest struct {
	RoleID    uint      `json:"role_id" binding:"required"`
	ProjectID uint      `json:"project_id"`
	ExpiresAt string    `json:"expires_at"`
}

// CreateRole 创建角色
func CreateRole(req *CreateRoleRequest) error {
	var existing model.Role
	if err := model.DB.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		return errors.New("角色名称已存在")
	}

	role := model.Role{
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		ParentID:    req.ParentID,
	}

	if req.Type == "" {
		role.Type = "custom"
	}

	if err := model.DB.Create(&role).Error; err != nil {
		return err
	}

	if len(req.Permissions) > 0 {
		for _, perm := range req.Permissions {
			parts := parsePermission(perm)
			if len(parts) == 2 {
				actions, _ := json.Marshal([]string{})
				actions, _ = json.Marshal(parseActions(parts[1]))
				rolePermission := model.RolePermission{
					RoleID:   role.ID,
					Resource: parts[0],
					Actions:  string(actions),
				}
				model.DB.Create(&rolePermission)
			}
		}
	}

	return nil
}

// GetRole 获取角色详情
func GetRole(id uint) (*model.Role, error) {
	var role model.Role
	if err := model.DB.Preload("Permissions").First(&role, id).Error; err != nil {
		return nil, errors.New("角色不存在")
	}
	return &role, nil
}

// ListRoles 获取角色列表
func ListRoles(page, pageSize int) (int64, []model.Role, error) {
	var total int64
	var roles []model.Role

	model.DB.Model(&model.Role{}).Count(&total)
	model.DB.Preload("Permissions").Offset((page - 1) * pageSize).Limit(pageSize).Find(&roles)

	return total, roles, nil
}

// UpdateRole 更新角色
func UpdateRole(id uint, req *UpdateRoleRequest) error {
	var role model.Role
	if err := model.DB.First(&role, id).Error; err != nil {
		return errors.New("角色不存在")
	}

	if role.Type == "system" {
		return errors.New("系统角色不能修改")
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		var existing model.Role
		if err := model.DB.Where("name = ? AND id != ?", *req.Name, id).First(&existing).Error; err == nil {
			return errors.New("角色名称已存在")
		}
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.ParentID != nil {
		updates["parent_id"] = *req.ParentID
	}

	if len(updates) > 0 {
		if err := model.DB.Model(&role).Updates(updates).Error; err != nil {
			return err
		}
	}

	if req.Permissions != nil {
		model.DB.Where("role_id = ?", id).Delete(&model.RolePermission{})

		for _, perm := range *req.Permissions {
			parts := parsePermission(perm)
			if len(parts) == 2 {
				actions, _ := json.Marshal(parseActions(parts[1]))
				rolePermission := model.RolePermission{
					RoleID:   id,
					Resource: parts[0],
					Actions:  string(actions),
				}
				model.DB.Create(&rolePermission)
			}
		}
	}

	return nil
}

// DeleteRole 删除角色
func DeleteRole(id uint) error {
	var role model.Role
	if err := model.DB.First(&role, id).Error; err != nil {
		return errors.New("角色不存在")
	}

	if role.Type == "system" {
		return errors.New("系统角色不能删除")
	}

	var count int64
	model.DB.Model(&model.UserRole{}).Where("role_id = ?", id).Count(&count)
	if count > 0 {
		return errors.New("该角色已被分配，不能删除")
	}

	return model.DB.Delete(&role).Error
}

// AssignRole 分配角色给用户
func AssignRole(userID uint, req *AssignRoleRequest) error {
	var role model.Role
	if err := model.DB.First(&role, req.RoleID).Error; err != nil {
		return errors.New("角色不存在")
	}

	var existing model.UserRole
	result := model.DB.Where("user_id = ? AND role_id = ? AND project_id = ?", userID, req.RoleID, req.ProjectID).First(&existing)
	if result.Error == nil {
		return errors.New("用户已拥有该角色")
	}

	userRole := model.UserRole{
		UserID:    userID,
		RoleID:    req.RoleID,
		ProjectID: req.ProjectID,
	}

	if req.ExpiresAt != "" {
		// 解析过期时间
	}

	return model.DB.Create(&userRole).Error
}

// RemoveRole 移除用户角色
func RemoveRole(userID uint, roleID uint) error {
	return model.DB.Where("user_id = ? AND role_id = ?", userID, roleID).Delete(&model.UserRole{}).Error
}

// GetUserRoles 获取用户角色列表
func GetUserRoles(userID uint) ([]model.UserRole, error) {
	var userRoles []model.UserRole
	err := model.DB.Preload("Role").Preload("Project").Where("user_id = ?", userID).Find(&userRoles).Error
	return userRoles, err
}

// CheckPermission 检查用户是否有权限
func CheckPermission(userID uint, resource string, action string) bool {
	var userRoles []model.UserRole
	model.DB.Preload("Role.Permissions").Where("user_id = ?", userID).Find(&userRoles)

	for _, ur := range userRoles {
		if ur.Role == nil {
			continue
		}

		if checkRolePermission(ur.Role, resource, action) {
			return true
		}

		if ur.Role.ParentID > 0 {
			var parentRole model.Role
			if model.DB.Preload("Permissions").First(&parentRole, ur.Role.ParentID).Error == nil {
				if checkRolePermission(&parentRole, resource, action) {
					return true
				}
			}
		}
	}

	return false
}

// checkRolePermission 检查角色权限
func checkRolePermission(role *model.Role, resource string, action string) bool {
	for _, perm := range role.Permissions {
		if perm.Resource == resource {
			var actions []string
			if err := json.Unmarshal([]byte(perm.Actions), &actions); err == nil {
				for _, a := range actions {
					if a == action || a == "*" {
						return true
					}
				}
			}
		}
	}
	return false
}

// parsePermission 解析权限字符串 "resource:actions"
func parsePermission(perm string) []string {
	for i, c := range perm {
		if c == ':' {
			return []string{perm[:i], perm[i+1:]}
		}
	}
	return []string{perm, ""}
}

// parseActions 解析动作字符串 "action1,action2"
func parseActions(actionsStr string) []string {
	var actions []string
	if actionsStr == "*" {
		return []string{"view", "create", "update", "delete", "approve", "export"}
	}
	start := 0
	for i, c := range actionsStr {
		if c == ',' {
			actions = append(actions, actionsStr[start:i])
			start = i + 1
		}
	}
	if start < len(actionsStr) {
		actions = append(actions, actionsStr[start:])
	}
	return actions
}