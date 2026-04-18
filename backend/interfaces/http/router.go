/**
 * HTTP Router
 * 配置 HTTP 路由 (Gin 框架)
 */
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupRoutes 设置路由
func SetupRoutes() *gin.Engine {
	return SetupRoutesWithManagement(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
}

func SetupRoutesWithUsers(userHandler *UserHandler) *gin.Engine {
	return SetupRoutesWithManagement(userHandler, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
}

func SetupRoutesWithManagement(
	userHandler *UserHandler,
	agentHandler *AgentHandler,
	providerHandler *LLMProviderHandler,
	channelHandler *ChannelHandler,
	sessionHandler *SessionHandler,
	conversationRecordHandler *ConversationRecordHandler,
	authHandler *AuthHandler,
	mcpHandler *MCPHandler,
	skillHandler *SkillHandler,
	projectHandler *ProjectHandler,
	requirementHandler *RequirementHandler,
	stateMachineHandler *StateMachineHandler,
	projectStateMachineHandler *ProjectStateMachineHandler,
	requirementTypeHandler *RequirementTypeHandler,
	heartbeatHandler *HeartbeatHandler,
	heartbeatTemplateHandler *HeartbeatTemplateHandler,
	heartbeatScenarioHandler *HeartbeatScenarioHandler,
) *gin.Engine {
	engine := gin.Default()

	// 认证中间件
	requireAuth := func(c *gin.Context) {
		if authHandler == nil {
			c.Next()
			return
		}
		if _, err := authHandler.Authorize(c.Request); err != nil {
			c.JSON(http.StatusUnauthorized, HTTPError{Code: http.StatusUnauthorized, Message: "unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}

	v1 := engine.Group("/api/v1")

	if authHandler != nil {
		auth := v1.Group("/auth")
		auth.POST("/login", authHandler.Login)
		auth.GET("/me", requireAuth, authHandler.Me)

		// Token管理路由
		usersTokens := v1.Group("/users/tokens", requireAuth)
		usersTokens.POST("", authHandler.CreateToken)
		usersTokens.GET("", authHandler.ListTokens)
		usersTokens.DELETE("/:id", authHandler.DeleteToken)
	}

	// MCP 路由
	if mcpHandler != nil {
		mcpServers := v1.Group("/mcp/servers", requireAuth)
		mcpServers.POST("", mcpHandler.CreateServer)
		mcpServers.GET("", mcpHandler.handleGetServers)
		mcpServers.PUT("", mcpHandler.UpdateServer)
		mcpServers.DELETE("", mcpHandler.DeleteServer)
		mcpServers.POST("/test", mcpHandler.TestServer)
		mcpServers.POST("/refresh", mcpHandler.RefreshCapabilities)
		mcpServers.GET("/tools", mcpHandler.ListTools)

		mcpBindings := v1.Group("/mcp/bindings", requireAuth)
		mcpBindings.GET("", mcpHandler.ListBindings)
		mcpBindings.POST("", mcpHandler.CreateBinding)
		mcpBindings.PUT("", mcpHandler.UpdateBinding)
		mcpBindings.DELETE("", mcpHandler.DeleteBinding)
	}

	if userHandler != nil {
		users := v1.Group("/users")
		users.POST("", userHandler.CreateUser)
		users.GET("", requireAuth, userHandler.handleGetUsers)
		users.PUT("", requireAuth, userHandler.UpdateUser)
		users.DELETE("", requireAuth, userHandler.DeleteUser)
	}

	if agentHandler != nil {
		agents := v1.Group("/agents", requireAuth)
		agents.POST("", agentHandler.CreateAgent)
		agents.GET("", agentHandler.handleGetAgents)
		agents.PUT("", agentHandler.UpdateAgent)
		agents.PATCH("", agentHandler.PatchAgent)
		agents.DELETE("", agentHandler.DeleteAgent)
	}

	if providerHandler != nil {
		providers := v1.Group("/providers", requireAuth)
		providers.POST("", providerHandler.CreateProvider)
		providers.GET("", providerHandler.handleGetProviders)
		providers.PUT("", providerHandler.UpdateProvider)
		providers.DELETE("", providerHandler.DeleteProvider)
		providers.POST("/test", providerHandler.TestConnection)
	}

	if channelHandler != nil {
		channels := v1.Group("/channels", requireAuth)
		channels.POST("", channelHandler.CreateChannel)
		channels.GET("", channelHandler.handleGetChannels)
		channels.PUT("", channelHandler.UpdateChannel)
		channels.DELETE("", channelHandler.DeleteChannel)
		channels.GET("/types", channelHandler.ListChannelTypes)
	}

	if sessionHandler != nil {
		sessions := v1.Group("/sessions", requireAuth)
		sessions.POST("", sessionHandler.CreateSession)
		sessions.GET("", sessionHandler.handleGetSessions)
		sessions.DELETE("", sessionHandler.DeleteSession)

		// 子路由：带路径参数
		sessions.GET("/:sessionKey", sessionHandler.GetSession)
		sessions.DELETE("/:sessionKey", sessionHandler.DeleteSessionByPath)
		sessions.POST("/:sessionKey/touch", sessionHandler.TouchSession)
		sessions.GET("/:sessionKey/metadata", sessionHandler.GetSessionMetadata)
		sessions.PUT("/:sessionKey/metadata", sessionHandler.UpdateSessionMetadata)
	}

	if conversationRecordHandler != nil {
		records := v1.Group("/conversation-records", requireAuth)
		records.POST("", conversationRecordHandler.CreateRecord)
		records.GET("", conversationRecordHandler.handleGetRecords)
		records.GET("/session/:sessionKey", conversationRecordHandler.GetRecordsBySession)
		records.GET("/trace/:traceId", conversationRecordHandler.GetRecordsByTrace)
		records.GET("/stats", conversationRecordHandler.GetStats)
	}

	if projectHandler != nil {
		projects := v1.Group("/projects", requireAuth)
		projects.POST("", projectHandler.CreateProject)
		projects.GET("", projectHandler.handleGetProjects)
		projects.PUT("", projectHandler.UpdateProject)
		projects.DELETE("", projectHandler.DeleteProject)
	}

	if heartbeatHandler != nil {
		heartbeats := v1.Group("/heartbeats", requireAuth)
		heartbeats.GET("", heartbeatHandler.ListHeartbeats)
		heartbeats.POST("", heartbeatHandler.CreateHeartbeat)
		heartbeats.GET("/:id", heartbeatHandler.GetHeartbeat)
		heartbeats.PUT("/:id", heartbeatHandler.UpdateHeartbeat)
		heartbeats.DELETE("/:id", heartbeatHandler.DeleteHeartbeat)
		heartbeats.POST("/:id/trigger", heartbeatHandler.TriggerHeartbeat)
	}

	if heartbeatTemplateHandler != nil {
		templates := v1.Group("/heartbeat-templates", requireAuth)
		templates.GET("", heartbeatTemplateHandler.ListTemplates)
		templates.POST("", heartbeatTemplateHandler.CreateTemplate)
		templates.DELETE("/:id", heartbeatTemplateHandler.DeleteTemplate)
	}

	if heartbeatScenarioHandler != nil {
		scenarios := v1.Group("/heartbeat-scenarios", requireAuth)
		scenarios.GET("", heartbeatScenarioHandler.ListScenarios)
		scenarios.POST("", heartbeatScenarioHandler.CreateScenario)
		scenarios.GET("/:code", heartbeatScenarioHandler.GetScenario)
		scenarios.PUT("/:code", heartbeatScenarioHandler.UpdateScenario)
		scenarios.DELETE("/:id", heartbeatScenarioHandler.DeleteScenario)
	}

	if projectHandler != nil && heartbeatScenarioHandler != nil {
		v1.POST("/projects/:project_id/apply-scenario", requireAuth, func(c *gin.Context) {
			c.Set("projectService", projectHandler.ProjectService())
			heartbeatScenarioHandler.ApplyScenarioToProject(c)
		})
	}

	if requirementHandler != nil {
		requirements := v1.Group("/requirements", requireAuth)
		requirements.POST("", requirementHandler.CreateRequirement)
		requirements.GET("", requirementHandler.handleGetRequirements)
		requirements.PUT("", requirementHandler.UpdateRequirement)
		requirements.DELETE("", requirementHandler.DeleteRequirement)
		requirements.POST("/dispatch", requirementHandler.DispatchRequirement)
		requirements.POST("/pr", requirementHandler.ReportRequirementPROpened)
		requirements.POST("/redispatch", requirementHandler.RedispatchRequirement)
		requirements.POST("/copy-and-dispatch", requirementHandler.CopyAndDispatchRequirement)
		requirements.POST("/reset", requirementHandler.RedispatchRequirement)
		requirements.POST("/batch-delete", requirementHandler.BatchDeleteRequirements)
		requirements.PUT("/status", requirementHandler.UpdateRequirementStatus)
		requirements.GET("/transition-history", requirementHandler.GetRequirementTransitionHistory)
		requirements.GET("/status-stats", requirementHandler.GetStatusStats)
	}

	// 需求类型路由
	if requirementTypeHandler != nil {
		reqTypes := v1.Group("/requirement-types", requireAuth)
		reqTypes.GET("", requirementTypeHandler.ListRequirementTypes)
		reqTypes.POST("", requirementTypeHandler.CreateRequirementType)
		reqTypes.DELETE("", requirementTypeHandler.DeleteRequirementType)
	}

	// OpenCode 路由
	v1.GET("/opencode/models", requireAuth, ListOpenCodeModels)

	// 工具路由
	toolsHandler := NewToolsHandler()
	v1.GET("/tools/builtin", toolsHandler.ListBuiltInTools)

	// Skill 路由
	if skillHandler != nil {
		skills := v1.Group("/skills", requireAuth)
		skills.GET("", skillHandler.ListSkills)
		skills.GET("/detail", skillHandler.GetSkill)
		skills.GET("/simple", skillHandler.ListSkillsSimple)
	}

	// 状态机路由
	if stateMachineHandler != nil {
		stateMachines := v1.Group("/state-machines", requireAuth)
		stateMachines.GET("", stateMachineHandler.ListStateMachines)
		stateMachines.POST("", stateMachineHandler.CreateStateMachine)
		stateMachines.GET("/:id", stateMachineHandler.GetStateMachine)
		stateMachines.PUT("/:id", stateMachineHandler.UpdateStateMachine)
		stateMachines.DELETE("/:id", stateMachineHandler.DeleteStateMachine)

		// 状态统计
		v1.GET("/requirements/states/summary", requireAuth, stateMachineHandler.GetStateSummary)

		// 需求状态转换
		v1.POST("/requirements/:requirement_id/transitions", requireAuth, stateMachineHandler.TriggerTransition)
		v1.GET("/requirements/:requirement_id/state", requireAuth, stateMachineHandler.GetRequirementState)
		v1.POST("/requirements/:requirement_id/state", requireAuth, stateMachineHandler.InitializeRequirementState)
		v1.GET("/requirements/:requirement_id/transitions/history", requireAuth, stateMachineHandler.GetTransitionHistory)
	}

	// 项目状态机关联路由
	if projectStateMachineHandler != nil {
		v1.GET("/project-state-machines/requirement-types", requireAuth, projectStateMachineHandler.GetAvailableRequirementTypes)
		v1.GET("/projects/:project_id/state-machines", requireAuth, projectStateMachineHandler.ListProjectStateMachines)
		v1.POST("/projects/:project_id/state-machines", requireAuth, projectStateMachineHandler.SetProjectStateMachine)
		v1.GET("/projects/:project_id/state-machines/:requirement_type", requireAuth, projectStateMachineHandler.GetProjectStateMachineByType)
		v1.DELETE("/project-state-machines/:id", requireAuth, projectStateMachineHandler.DeleteProjectStateMachine)
	}

	return engine
}

// RegisterWebhookRoutes 注册 Webhook 相关路由
func RegisterWebhookRoutes(engine *gin.Engine, webhookHandler *WebhookHandler, githubWebhookHandler *GitHubWebhookHandler, authHandler *AuthHandler) {
	if webhookHandler == nil || githubWebhookHandler == nil {
		return
	}

	// API v1 前缀组
	v1 := engine.Group("/api/v1")

	// 认证中间件
	requireAuth := func(c *gin.Context) {
		if authHandler == nil {
			c.Next()
			return
		}
		if _, err := authHandler.Authorize(c.Request); err != nil {
			c.JSON(http.StatusUnauthorized, HTTPError{Code: http.StatusUnauthorized, Message: "unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}

	// Webhook 接收端点（无需认证，统一 /api/v1 前缀）
	// 语义化 URL：/api/v1/webhook/repos/{owner}/{repo}
	v1.POST("/webhook/repos/:owner/:repo", webhookHandler.HandleWebhookByRepo)
	// 通用端点，通过 payload 中的 repo 匹配项目
	v1.POST("/webhook", webhookHandler.HandleWebhook)
	// 内部接口：更新所有启用的 webhook URL（供 tunnel start 调用，需认证）
	v1.POST("/internal/webhooks/update-all", requireAuth, func(c *gin.Context) {
		githubWebhookHandler.UpdateAllWebhooksIfNeeded()
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	// GitHub Webhook 管理 API（需认证）
	webhooks := v1.Group("/github-webhooks", requireAuth)
	webhooks.GET("/configs", githubWebhookHandler.ListConfigs)
	webhooks.POST("/configs", githubWebhookHandler.CreateConfig)
	webhooks.PUT("/configs/:id", githubWebhookHandler.UpdateConfig)
	webhooks.DELETE("/configs/:id", githubWebhookHandler.DeleteConfig)
	webhooks.POST("/configs/:id/enable", githubWebhookHandler.EnableWebhook)
	webhooks.POST("/configs/:id/disable", githubWebhookHandler.DisableWebhook)
	webhooks.GET("/configs/:id/status", githubWebhookHandler.GetWebhookStatus)
	webhooks.GET("/configs/:id/check-url", githubWebhookHandler.CheckWebhookURL)
	webhooks.POST("/configs/:id/update-url", githubWebhookHandler.UpdateWebhookURL)
	webhooks.GET("/configs/:id/event-logs", githubWebhookHandler.ListEventLogs)
	webhooks.DELETE("/configs/:id/event-logs", githubWebhookHandler.ClearEventLogs)
	webhooks.GET("/configs/:id/bindings", githubWebhookHandler.ListBindings)
	webhooks.POST("/bindings", githubWebhookHandler.CreateBinding)
	webhooks.DELETE("/bindings/:id", githubWebhookHandler.DeleteBinding)
	webhooks.GET("/heartbeats", githubWebhookHandler.ListHeartbeats)
}
