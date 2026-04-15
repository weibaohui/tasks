package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

type ProjectHandler struct {
	projectService *application.ProjectApplicationService
}

func NewProjectHandler(projectService *application.ProjectApplicationService) *ProjectHandler {
	return &ProjectHandler{projectService: projectService}
}

type CreateProjectRequest struct {
	Name          string   `json:"name"`
	GitRepoURL    string   `json:"git_repo_url"`
	DefaultBranch string   `json:"default_branch"`
	InitSteps     []string `json:"init_steps"`
}

type UpdateProjectRequest struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	GitRepoURL          string   `json:"git_repo_url"`
	DefaultBranch       string   `json:"default_branch"`
	InitSteps           []string `json:"init_steps"`
	DispatchChannelCode *string  `json:"dispatch_channel_code,omitempty"`
	DispatchSessionKey  *string  `json:"dispatch_session_key,omitempty"`
	MaxConcurrentAgents *int     `json:"max_concurrent_agents,omitempty"`
}

func (h *ProjectHandler) CreateProject(c *gin.Context) {
	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	project, err := h.projectService.CreateProject(c.Request.Context(), application.CreateProjectCommand{
		Name:          req.Name,
		GitRepoURL:    req.GitRepoURL,
		DefaultBranch: req.DefaultBranch,
		InitSteps:     req.InitSteps,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, projectToMap(project))
}

func (h *ProjectHandler) GetProject(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	project, err := h.projectService.GetProject(c.Request.Context(), domain.NewProjectID(id))
	if err != nil {
		c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, projectToMap(project))
}

func (h *ProjectHandler) ListProjects(c *gin.Context) {
	projects, err := h.projectService.ListProjects(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	resp := make([]map[string]interface{}, 0, len(projects))
	for _, project := range projects {
		resp = append(resp, projectToMap(project))
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	var req UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
		return
	}
	project, err := h.projectService.UpdateProject(c.Request.Context(), application.UpdateProjectCommand{
		ID:                  domain.NewProjectID(req.ID),
		Name:                req.Name,
		GitRepoURL:          req.GitRepoURL,
		DefaultBranch:       req.DefaultBranch,
		InitSteps:           req.InitSteps,
		DispatchChannelCode: req.DispatchChannelCode,
		DispatchSessionKey:  req.DispatchSessionKey,
		MaxConcurrentAgents: req.MaxConcurrentAgents,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, projectToMap(project))
}

func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "id is required"})
		return
	}
	if err := h.projectService.DeleteProject(c.Request.Context(), domain.NewProjectID(id)); err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"message": "ok"})
}

// handleGetProjects 根据 query 参数分发到 GetProject 或 ListProjects
func (h *ProjectHandler) handleGetProjects(c *gin.Context) {
	if c.Query("id") != "" {
		h.GetProject(c)
		return
	}
	h.ListProjects(c)
}

func projectToMap(project *domain.Project) map[string]interface{} {
	return map[string]interface{}{
		"id":                  project.ID().String(),
		"name":                project.Name(),
		"git_repo_url":        project.GitRepoURL(),
		"default_branch":      project.DefaultBranch(),
		"init_steps":          project.InitSteps(),
		"dispatch_channel_code": project.DispatchChannelCode(),
		"dispatch_session_key":  project.DispatchSessionKey(),
		"max_concurrent_agents": project.MaxConcurrentAgents(),
		"created_at":          project.CreatedAt().UnixMilli(),
		"updated_at":          project.UpdatedAt().UnixMilli(),
	}
}
