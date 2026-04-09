package http

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

type ProjectHandler struct {
	projectService     *application.ProjectApplicationService
	heartbeatScheduler *application.HeartbeatScheduler
}

func NewProjectHandler(projectService *application.ProjectApplicationService, heartbeatScheduler *application.HeartbeatScheduler) *ProjectHandler {
	return &ProjectHandler{projectService: projectService, heartbeatScheduler: heartbeatScheduler}
}

type CreateProjectRequest struct {
	Name          string   `json:"name"`
	GitRepoURL    string   `json:"git_repo_url"`
	DefaultBranch string   `json:"default_branch"`
	InitSteps     []string `json:"init_steps"`
}

type UpdateProjectRequest struct {
	ID                       string   `json:"id"`
	Name                     string   `json:"name"`
	GitRepoURL               string   `json:"git_repo_url"`
	DefaultBranch            string   `json:"default_branch"`
	InitSteps                []string `json:"init_steps"`
	HeartbeatEnabled         *bool    `json:"heartbeat_enabled,omitempty"`
	HeartbeatIntervalMinutes *int     `json:"heartbeat_interval_minutes,omitempty"`
	HeartbeatMDContent       *string  `json:"heartbeat_md_content,omitempty"`
	AgentCode                *string  `json:"agent_code,omitempty"`
	DispatchChannelCode      *string  `json:"dispatch_channel_code,omitempty"`
	DispatchSessionKey       *string  `json:"dispatch_session_key,omitempty"`
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
		ID:                       domain.NewProjectID(req.ID),
		Name:                     req.Name,
		GitRepoURL:               req.GitRepoURL,
		DefaultBranch:            req.DefaultBranch,
		InitSteps:                req.InitSteps,
		HeartbeatEnabled:         req.HeartbeatEnabled,
		HeartbeatIntervalMinutes: req.HeartbeatIntervalMinutes,
		HeartbeatMDContent:       req.HeartbeatMDContent,
		AgentCode:                req.AgentCode,
		DispatchChannelCode:      req.DispatchChannelCode,
		DispatchSessionKey:       req.DispatchSessionKey,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	// 刷新心跳调度
	if h.heartbeatScheduler != nil {
		if err := h.heartbeatScheduler.RefreshSchedule(c.Request.Context()); err != nil {
			log.Printf("failed to refresh heartbeat schedule: %v", err)
		}
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
		"id":                        project.ID().String(),
		"name":                      project.Name(),
		"git_repo_url":              project.GitRepoURL(),
		"default_branch":            project.DefaultBranch(),
		"init_steps":                project.InitSteps(),
		"heartbeat_enabled":         project.HeartbeatEnabled(),
		"heartbeat_interval_minutes": project.HeartbeatIntervalMinutes(),
		"heartbeat_md_content":      project.HeartbeatMDContent(),
		"agent_code":                project.AgentCode(),
		"dispatch_channel_code":     project.DispatchChannelCode(),
		"dispatch_session_key":      project.DispatchSessionKey(),
		"created_at":                project.CreatedAt().UnixMilli(),
		"updated_at":                project.UpdatedAt().UnixMilli(),
	}
}
