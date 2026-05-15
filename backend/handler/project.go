package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"k8sgate/k8s"
	"k8sgate/pkg"
	"k8sgate/service"
)

type ProjectHandler struct{}

func NewProjectHandler() *ProjectHandler {
	return &ProjectHandler{}
}

func (h *ProjectHandler) List(c *gin.Context) {
	var q service.ProjectListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}

	total, projects, err := service.GetProjectList(&q)
	if err != nil {
		pkg.Fail(c, http.StatusInternalServerError, 50001, "查询失败")
		return
	}
	pkg.OK(c, pkg.PageData(total, projects))
}

func (h *ProjectHandler) Create(c *gin.Context) {
	var req service.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误: "+err.Error())
		return
	}

	project, err := service.CreateProject(&req)
	if err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	pkg.OK(c, project)
}

func (h *ProjectHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "无效的项目ID")
		return
	}

	project, err := service.GetProject(uint(id))
	if err != nil {
		pkg.Fail(c, http.StatusNotFound, 40001, err.Error())
		return
	}
	pkg.OK(c, project)
}

func (h *ProjectHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "无效的项目ID")
		return
	}

	var req service.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}

	if err := service.UpdateProject(uint(id), &req); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	pkg.OKMsg(c, "更新成功")
}

func (h *ProjectHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "无效的项目ID")
		return
	}

	if err := service.DeleteProject(uint(id)); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	pkg.OKMsg(c, "删除成功")
}

func (h *ProjectHandler) AssignUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "无效的项目ID")
		return
	}

	var req service.AssignUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "参数错误: "+err.Error())
		return
	}

	if err := service.AssignUserToProject(uint(id), &req); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	pkg.OKMsg(c, "分配成功")
}

func (h *ProjectHandler) RemoveUser(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "无效的项目ID")
		return
	}
	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, "无效的用户ID")
		return
	}

	if err := service.RemoveUserFromProject(uint(projectID), uint(userID)); err != nil {
		pkg.Fail(c, http.StatusBadRequest, 40001, err.Error())
		return
	}
	pkg.OKMsg(c, "移除成功")
}

func (h *ProjectHandler) ListNamespaces(c *gin.Context) {
	namespaces, err := k8s.ListNamespaces()
	if err != nil {
		pkg.Fail(c, http.StatusInternalServerError, 50002, "获取命名空间失败: "+err.Error())
		return
	}
	pkg.OK(c, namespaces)
}
