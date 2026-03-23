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
func SetupRoutes(handler *TaskHandler) *http.ServeMux {
	return SetupRoutesWithUsers(handler, nil)
}

func SetupRoutesWithUsers(handler *TaskHandler, userHandler *UserHandler) *http.ServeMux {
	return SetupRoutesWithManagement(handler, userHandler, nil, nil, nil, nil, nil, nil, nil)
}

func SetupRoutesWithManagement(
	handler *TaskHandler,
	userHandler *UserHandler,
	agentHandler *AgentHandler,
	providerHandler *LLMProviderHandler,
	channelHandler *ChannelHandler,
	sessionHandler *SessionHandler,
	conversationRecordHandler *ConversationRecordHandler,
	authHandler *AuthHandler,
	mcpHandler *MCPHandler,
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

	// POST /api/v1/tasks - 创建任务
	mux.HandleFunc("/api/v1/tasks", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handler.CreateTask(w, r)
		case http.MethodGet:
			// GET /api/v1/tasks?id=xxx - 获取单个任务
			handler.GetTask(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/v1/tasks/clear", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handler.ClearAllTasks(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}))

	mux.HandleFunc("/api/v1/tasks/all", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handler.ListAllTasks(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}))

	// GET /api/v1/tasks/trace/{trace_id} - 获取任务列表（按 trace_id）
	mux.HandleFunc("/api/v1/tasks/trace/", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handler.ListTasksByTrace(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	// GET /api/v1/traces/{trace_id}/tree - 获取任务树
	mux.HandleFunc("/api/v1/traces/", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			if strings.HasSuffix(r.URL.Path, "/tree") {
				handler.GetTaskTree(w, r)
			} else {
				http.Error(w, "not found", http.StatusNotFound)
			}
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	// POST /api/v1/tasks/{id}/cancel - 取消任务
	// POST /api/v1/tasks/{id}/start - 启动任务
	mux.HandleFunc("/api/v1/tasks/", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if r.Method == http.MethodPost {
			if strings.HasSuffix(path, "/cancel") {
				handler.CancelTask(w, r)
				return
			}
			if strings.HasSuffix(path, "/start") {
				handler.StartTask(w, r)
				return
			}
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}))

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

		mux.HandleFunc("/api/v1/providers/embedding", requireAuth(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				providerHandler.GetEmbeddingModels(w, r)
			case http.MethodPut:
				providerHandler.UpdateEmbeddingModels(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
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
	}

	return mux
}
