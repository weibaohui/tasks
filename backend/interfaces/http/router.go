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

	return mux
}
