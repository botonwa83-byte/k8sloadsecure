package service

import (
	"errors"

	"k8sgate/model"
)

type CreateProjectRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Namespaces  []string `json:"namespaces" binding:"required,min=1"`
}

type UpdateProjectRequest struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Namespaces  []string `json:"namespaces"`
}

type ProjectListQuery struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
	Keyword  string `form:"keyword"`
}

type AssignUserRequest struct {
	UserID     uint   `json:"user_id" binding:"required"`
	Permission string `json:"permission" binding:"required,oneof=read readwrite"`
}

func CreateProject(req *CreateProjectRequest) (*model.Project, error) {
	var count int64
	model.DB.Model(&model.Project{}).Where("name = ?", req.Name).Count(&count)
	if count > 0 {
		return nil, errors.New("项目名称已存在")
	}

	project := model.Project{
		Name:        req.Name,
		Description: req.Description,
	}
	if err := model.DB.Create(&project).Error; err != nil {
		return nil, errors.New("创建项目失败")
	}

	for _, ns := range req.Namespaces {
		model.DB.Create(&model.ProjectNamespace{
			ProjectID: project.ID,
			Namespace: ns,
		})
	}

	model.DB.Preload("Namespaces").First(&project, project.ID)
	return &project, nil
}

func GetProjectList(q *ProjectListQuery) (int64, []map[string]interface{}, error) {
	var total int64
	var projects []model.Project

	db := model.DB.Model(&model.Project{})
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		db = db.Where("name LIKE ? OR description LIKE ?", like, like)
	}

	db.Count(&total)

	offset := (q.Page - 1) * q.PageSize
	if offset < 0 {
		offset = 0
	}
	err := db.Order("id DESC").Offset(offset).Limit(q.PageSize).
		Preload("Namespaces").Find(&projects).Error
	if err != nil {
		return 0, nil, err
	}

	result := make([]map[string]interface{}, len(projects))
	for i, p := range projects {
		namespaces := make([]string, len(p.Namespaces))
		for j, ns := range p.Namespaces {
			namespaces[j] = ns.Namespace
		}

		var userCount int64
		model.DB.Model(&model.UserProject{}).Where("project_id = ?", p.ID).Count(&userCount)

		result[i] = map[string]interface{}{
			"id":          p.ID,
			"name":        p.Name,
			"description": p.Description,
			"namespaces":  namespaces,
			"user_count":  userCount,
			"created_at":  p.CreatedAt,
			"updated_at":  p.UpdatedAt,
		}
	}

	return total, result, nil
}

func GetProject(id uint) (*model.Project, error) {
	var project model.Project
	err := model.DB.Preload("Namespaces").Preload("Users.User").First(&project, id).Error
	if err != nil {
		return nil, errors.New("项目不存在")
	}
	return &project, nil
}

func UpdateProject(id uint, req *UpdateProjectRequest) error {
	var project model.Project
	if err := model.DB.First(&project, id).Error; err != nil {
		return errors.New("项目不存在")
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		var count int64
		model.DB.Model(&model.Project{}).Where("name = ? AND id != ?", *req.Name, id).Count(&count)
		if count > 0 {
			return errors.New("项目名称已存在")
		}
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}

	if len(updates) > 0 {
		model.DB.Model(&project).Updates(updates)
	}

	if req.Namespaces != nil {
		model.DB.Where("project_id = ?", id).Delete(&model.ProjectNamespace{})
		for _, ns := range req.Namespaces {
			model.DB.Create(&model.ProjectNamespace{
				ProjectID: id,
				Namespace: ns,
			})
		}
	}

	return nil
}

func DeleteProject(id uint) error {
	var project model.Project
	if err := model.DB.First(&project, id).Error; err != nil {
		return errors.New("项目不存在")
	}
	model.DB.Where("project_id = ?", id).Delete(&model.ProjectNamespace{})
	model.DB.Where("project_id = ?", id).Delete(&model.UserProject{})
	return model.DB.Delete(&project).Error
}

func AssignUserToProject(projectID uint, req *AssignUserRequest) error {
	var project model.Project
	if err := model.DB.First(&project, projectID).Error; err != nil {
		return errors.New("项目不存在")
	}

	var user model.User
	if err := model.DB.First(&user, req.UserID).Error; err != nil {
		return errors.New("用户不存在")
	}

	var existing model.UserProject
	result := model.DB.Where("user_id = ? AND project_id = ?", req.UserID, projectID).First(&existing)
	if result.Error == nil {
		return model.DB.Model(&existing).Update("permission", req.Permission).Error
	}

	return model.DB.Create(&model.UserProject{
		UserID:     req.UserID,
		ProjectID:  projectID,
		Permission: req.Permission,
	}).Error
}

func RemoveUserFromProject(projectID, userID uint) error {
	result := model.DB.Where("user_id = ? AND project_id = ?", userID, projectID).Delete(&model.UserProject{})
	if result.RowsAffected == 0 {
		return errors.New("用户未分配到该项目")
	}
	return nil
}
