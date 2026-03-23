/**
 * HTTP Router
 * 配置 HTTP 路由
 */
package http

import (
	"net/http"
	"strings"
)

// SetupRoutes 设置路由
// 注意：Go 标准库 http.ServeMux 不支持路径参数，路由按最长前缀匹配
func SetupRoutes(handler *TaskHandler) *http.ServeMux {
	return SetupRoutesWithUsers(handler, nil)
}

func SetupRoutesWithUsers(handler *TaskHandler, userHandler *UserHandler) *http.ServeMux {
	return SetupRoutesWithManagement(handler, userHandler, nil, nil, nil, nil)
}

func SetupRoutesWithManagement(
	handler *TaskHandler,
	userHandler *UserHandler,
	agentHandler *AgentHandler,
	providerHandler *LLMProviderHandler,
	channelHandler *ChannelHandler,
	sessionHandler *SessionHandler,
) *http.ServeMux {
	mux := http.NewServeMux()

	// POST /api/v1/tasks - 创建任务
	mux.HandleFunc("/api/v1/tasks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handler.CreateTask(w, r)
		case http.MethodGet:
			// GET /api/v1/tasks?id=xxx - 获取单个任务
			handler.GetTask(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/tasks/clear", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handler.ClearAllTasks(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/api/v1/tasks/all", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handler.ListAllTasks(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	// GET /api/v1/tasks/trace/{trace_id} - 获取任务列表（按 trace_id）
	mux.HandleFunc("/api/v1/tasks/trace/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handler.ListTasksByTrace(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// GET /api/v1/traces/{trace_id}/tree - 获取任务树
	mux.HandleFunc("/api/v1/traces/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			if strings.HasSuffix(r.URL.Path, "/tree") {
				handler.GetTaskTree(w, r)
			} else {
				http.Error(w, "not found", http.StatusNotFound)
			}
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// POST /api/v1/tasks/{id}/cancel - 取消任务
	// POST /api/v1/tasks/{id}/start - 启动任务
	mux.HandleFunc("/api/v1/tasks/", func(w http.ResponseWriter, r *http.Request) {
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
	})

	if userHandler != nil {
		mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
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
		mux.HandleFunc("/api/v1/agents", func(w http.ResponseWriter, r *http.Request) {
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
		})
	}

	if providerHandler != nil {
		mux.HandleFunc("/api/v1/providers", func(w http.ResponseWriter, r *http.Request) {
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
		})

		mux.HandleFunc("/api/v1/providers/test", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			providerHandler.TestConnection(w, r)
		})

		mux.HandleFunc("/api/v1/providers/embedding", func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				providerHandler.GetEmbeddingModels(w, r)
			case http.MethodPut:
				providerHandler.UpdateEmbeddingModels(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		})
	}

	if channelHandler != nil {
		mux.HandleFunc("/api/v1/channels", func(w http.ResponseWriter, r *http.Request) {
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
		})
	}

	if sessionHandler != nil {
		mux.HandleFunc("/api/v1/sessions", func(w http.ResponseWriter, r *http.Request) {
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
		})

		mux.HandleFunc("/api/v1/sessions/", func(w http.ResponseWriter, r *http.Request) {
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
		})
	}

	return mux
}
