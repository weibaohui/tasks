/**
 * HTTP Router
 * 配置 HTTP 路由
 */
package http

import (
	"encoding/json"
	"net/http"
	"strings"
)

// SetupRoutes 设置路由
// 注意：Go 标准库 http.ServeMux 不支持路径参数，路由按最长前缀匹配
func SetupRoutes() *http.ServeMux {
	return SetupRoutesWithManagement(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
}

func SetupRoutesWithUsers(userHandler *UserHandler) *http.ServeMux {
	return SetupRoutesWithManagement(userHandler, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
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
	hookHandler *HookHandler,
	stateMachineHandler *StateMachineHandler,
) *http.ServeMux {
	mux := http.NewServeMux()
	requireAuth := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if authHandler == nil {
				next(w, r)
				return
			}
			if _, err := authHandler.Authorize(r); err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusUnauthorized, Message: "unauthorized"})
				return
			}
			next(w, r)
		}
	}

	if authHandler != nil {
		mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			authHandler.Login(w, r)
		})
		mux.HandleFunc("/api/v1/auth/me", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			authHandler.Me(w, r)
		}))

		// Token管理路由
		mux.HandleFunc("/api/v1/users/tokens", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				authHandler.CreateToken(w, r)
			case http.MethodGet:
				authHandler.ListTokens(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}))
		mux.HandleFunc("/api/v1/users/tokens/", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				authHandler.DeleteToken(w, r)
			} else {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}))
	}

	// MCP 路由
	if mcpHandler != nil {
		mux.HandleFunc("/api/v1/mcp/servers", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				mcpHandler.CreateServer(w, r)
			case http.MethodGet:
				if r.URL.Query().Get("id") != "" {
					mcpHandler.GetServer(w, r)
					return
				}
				mcpHandler.ListServers(w, r)
			case http.MethodPut:
				mcpHandler.UpdateServer(w, r)
			case http.MethodDelete:
				mcpHandler.DeleteServer(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}))
		mux.HandleFunc("/api/v1/mcp/servers/test", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			mcpHandler.TestServer(w, r)
		}))
		mux.HandleFunc("/api/v1/mcp/servers/refresh", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			mcpHandler.RefreshCapabilities(w, r)
		}))
		mux.HandleFunc("/api/v1/mcp/servers/tools", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			mcpHandler.ListTools(w, r)
		}))
		mux.HandleFunc("/api/v1/mcp/bindings", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				mcpHandler.ListBindings(w, r)
			case http.MethodPost:
				mcpHandler.CreateBinding(w, r)
			case http.MethodPut:
				mcpHandler.UpdateBinding(w, r)
			case http.MethodDelete:
				mcpHandler.DeleteBinding(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}))
	}

	if userHandler != nil {
		mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost && authHandler != nil {
				if _, err := authHandler.Authorize(r); err != nil {
					w.WriteHeader(http.StatusUnauthorized)
					_ = json.NewEncoder(w).Encode(HTTPError{Code: http.StatusUnauthorized, Message: "unauthorized"})
					return
				}
			}
			switch r.Method {
			case http.MethodPost:
				userHandler.CreateUser(w, r)
			case http.MethodGet:
				if r.URL.Query().Get("id") != "" {
					userHandler.GetUser(w, r)
					return
				}
				userHandler.ListUsers(w, r)
			case http.MethodPut:
				userHandler.UpdateUser(w, r)
			case http.MethodDelete:
				userHandler.DeleteUser(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		})
	}

	if agentHandler != nil {
		mux.HandleFunc("/api/v1/agents", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				agentHandler.CreateAgent(w, r)
			case http.MethodGet:
				if r.URL.Query().Get("id") != "" || r.URL.Query().Get("code") != "" {
					agentHandler.GetAgent(w, r)
					return
				}
				agentHandler.ListAgents(w, r)
			case http.MethodPut:
				agentHandler.UpdateAgent(w, r)
			case http.MethodPatch:
				agentHandler.PatchAgent(w, r)
			case http.MethodDelete:
				agentHandler.DeleteAgent(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}))
	}

	if providerHandler != nil {
		mux.HandleFunc("/api/v1/providers", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				providerHandler.CreateProvider(w, r)
			case http.MethodGet:
				if r.URL.Query().Get("id") != "" {
					providerHandler.GetProvider(w, r)
					return
				}
				providerHandler.ListProviders(w, r)
			case http.MethodPut:
				providerHandler.UpdateProvider(w, r)
			case http.MethodDelete:
				providerHandler.DeleteProvider(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}))

		mux.HandleFunc("/api/v1/providers/test", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			providerHandler.TestConnection(w, r)
		}))
	}

	if channelHandler != nil {
		mux.HandleFunc("/api/v1/channels", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				channelHandler.CreateChannel(w, r)
			case http.MethodGet:
				if r.URL.Query().Get("id") != "" || r.URL.Query().Get("code") != "" {
					channelHandler.GetChannel(w, r)
					return
				}
				channelHandler.ListChannels(w, r)
			case http.MethodPut:
				channelHandler.UpdateChannel(w, r)
			case http.MethodDelete:
				channelHandler.DeleteChannel(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}))

		mux.HandleFunc("/api/v1/channels/types", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			channelHandler.ListChannelTypes(w, r)
		}))
	}

	if sessionHandler != nil {
		mux.HandleFunc("/api/v1/sessions", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				sessionHandler.CreateSession(w, r)
			case http.MethodGet:
				if r.URL.Query().Get("session_key") != "" {
					sessionHandler.GetSession(w, r)
					return
				}
				sessionHandler.ListSessions(w, r)
			case http.MethodDelete:
				sessionHandler.DeleteSession(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}))

		mux.HandleFunc("/api/v1/sessions/", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			switch {
			case strings.HasSuffix(path, "/touch"):
				if r.Method != http.MethodPost {
					http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
					return
				}
				sessionHandler.TouchSession(w, r)
			case strings.HasSuffix(path, "/metadata"):
				switch r.Method {
				case http.MethodGet:
					sessionHandler.GetSessionMetadata(w, r)
				case http.MethodPut:
					sessionHandler.UpdateSessionMetadata(w, r)
				default:
					http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				}
			default:
				switch r.Method {
				case http.MethodGet:
					sessionHandler.GetSession(w, r)
				case http.MethodDelete:
					sessionHandler.DeleteSession(w, r)
				default:
					http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				}
			}
		}))
	}

	if conversationRecordHandler != nil {
		mux.HandleFunc("/api/v1/conversation-records", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				conversationRecordHandler.CreateRecord(w, r)
			case http.MethodGet:
				if r.URL.Query().Get("id") != "" {
					conversationRecordHandler.GetRecord(w, r)
					return
				}
				conversationRecordHandler.ListRecords(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}))

		// GET /api/v1/conversation-records/session/{sessionKey}
		mux.HandleFunc("/api/v1/conversation-records/session/", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			sessionKey := strings.TrimPrefix(r.URL.Path, "/api/v1/conversation-records/session/")
			conversationRecordHandler.GetRecordsBySession(w, r, sessionKey)
		}))

		// GET /api/v1/conversation-records/trace/{traceId}
		mux.HandleFunc("/api/v1/conversation-records/trace/", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			traceId := strings.TrimPrefix(r.URL.Path, "/api/v1/conversation-records/trace/")
			conversationRecordHandler.GetRecordsByTrace(w, r, traceId)
		}))

		// GET /api/v1/conversation-records/stats
		mux.HandleFunc("/api/v1/conversation-records/stats", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			conversationRecordHandler.GetStats(w, r)
		}))
	}

	if projectHandler != nil {
		mux.HandleFunc("/api/v1/projects", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				projectHandler.CreateProject(w, r)
			case http.MethodGet:
				if r.URL.Query().Get("id") != "" {
					projectHandler.GetProject(w, r)
					return
				}
				projectHandler.ListProjects(w, r)
			case http.MethodPut:
				projectHandler.UpdateProject(w, r)
			case http.MethodDelete:
				projectHandler.DeleteProject(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}))
	}

	if requirementHandler != nil {
		mux.HandleFunc("/api/v1/requirements/dispatch", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			requirementHandler.DispatchRequirement(w, r)
		}))
		mux.HandleFunc("/api/v1/requirements/pr", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			requirementHandler.ReportRequirementPROpened(w, r)
		}))
		mux.HandleFunc("/api/v1/requirements/redispatch", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			requirementHandler.RedispatchRequirement(w, r)
		}))
		// 复制需求并派发新副本
		mux.HandleFunc("/api/v1/requirements/copy-and-dispatch", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			requirementHandler.CopyAndDispatchRequirement(w, r)
		}))
		// 重置需求 - 复用 RedispatchRequirement handler（语义相同）
		mux.HandleFunc("/api/v1/requirements/reset", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			requirementHandler.RedispatchRequirement(w, r)
		}))
		// 批量删除需求
		mux.HandleFunc("/api/v1/requirements/batch-delete", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			requirementHandler.BatchDeleteRequirements(w, r)
		}))
		mux.HandleFunc("/api/v1/requirements", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				requirementHandler.CreateRequirement(w, r)
			case http.MethodGet:
				if r.URL.Query().Get("id") != "" {
					requirementHandler.GetRequirement(w, r)
					return
				}
				requirementHandler.ListRequirements(w, r)
			case http.MethodPut:
				requirementHandler.UpdateRequirement(w, r)
			case http.MethodDelete:
				requirementHandler.DeleteRequirement(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}))
	}

	// Hook 配置路由
	if hookHandler != nil {
		mux.HandleFunc("/api/v1/hook-configs", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				hookHandler.CreateHookConfig(w, r)
			case http.MethodGet:
				if r.URL.Query().Get("id") != "" {
					hookHandler.GetHookConfig(w, r)
					return
				}
				hookHandler.ListHookConfigs(w, r)
			case http.MethodPut:
				hookHandler.UpdateHookConfig(w, r)
			case http.MethodDelete:
				hookHandler.DeleteHookConfig(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}))

		mux.HandleFunc("/api/v1/hook-configs/", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if strings.HasSuffix(path, "/enable") {
				if r.Method == http.MethodPatch {
					hookHandler.EnableHookConfig(w, r)
					return
				}
			} else if strings.HasSuffix(path, "/disable") {
				if r.Method == http.MethodPatch {
					hookHandler.DisableHookConfig(w, r)
					return
				}
			}
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}))

		// Hook 日志路由
		mux.HandleFunc("/api/v1/hook-logs", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				hookHandler.ListHookLogs(w, r)
				return
			}
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}))
	}

	// 工具路由
	toolsHandler := NewToolsHandler()
	mux.HandleFunc("/api/v1/tools/builtin", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		toolsHandler.ListBuiltInTools(w, r)
	})

	// Skill 路由
	if skillHandler != nil {
		mux.HandleFunc("/api/v1/skills", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				skillHandler.ListSkills(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}))
		mux.HandleFunc("/api/v1/skills/detail", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			skillHandler.GetSkill(w, r)
		}))
		mux.HandleFunc("/api/v1/skills/simple", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			skillHandler.ListSkillsSimple(w, r)
		}))
	}

	// 状态机路由
	if stateMachineHandler != nil {
		// 项目状态机管理
		mux.HandleFunc("/api/v1/projects/{project_id}/state-machines", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				stateMachineHandler.ListStateMachines(w, r)
			case http.MethodPost:
				stateMachineHandler.CreateStateMachine(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}))

		// 项目状态统计
		mux.HandleFunc("/api/v1/projects/{project_id}/requirements/states/summary", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				stateMachineHandler.GetProjectStateSummary(w, r)
			} else {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}))

		// 单个状态机操作
		mux.HandleFunc("/api/v1/state-machines/{id}", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				stateMachineHandler.GetStateMachine(w, r)
			case http.MethodPut:
				stateMachineHandler.UpdateStateMachine(w, r)
			case http.MethodDelete:
				stateMachineHandler.DeleteStateMachine(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}))

		// 类型绑定
		mux.HandleFunc("/api/v1/state-machines/{id}/bind", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				stateMachineHandler.BindType(w, r)
			} else {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}))

		mux.HandleFunc("/api/v1/state-machines/{id}/bind/{type}", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				stateMachineHandler.UnbindType(w, r)
			} else {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}))

		// 需求状态转换
		mux.HandleFunc("/api/v1/requirements/{requirement_id}/transitions", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				stateMachineHandler.TriggerTransition(w, r)
			} else {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}))

		// 需求当前状态
		mux.HandleFunc("/api/v1/requirements/{requirement_id}/state", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				stateMachineHandler.GetRequirementState(w, r)
			} else {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}))

		// 需求转换历史
		mux.HandleFunc("/api/v1/requirements/{requirement_id}/transitions/history", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				stateMachineHandler.GetTransitionHistory(w, r)
			} else {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		}))
	}

	return mux
}
