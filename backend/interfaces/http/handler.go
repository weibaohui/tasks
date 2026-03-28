/**
 * HTTP API Handler
 * 处理 HTTP 请求
 */
package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
)

// TaskHandler HTTP处理器
type TaskHandler struct {
	taskService  *application.TaskApplicationService
	queryService *application.QueryService
}

// NewTaskHandler 创建HTTP处理器
func NewTaskHandler(
	taskService *application.TaskApplicationService,
	queryService *application.QueryService,
) *TaskHandler {
	return &TaskHandler{
		taskService:  taskService,
		queryService: queryService,
	}
}

// HTTPError HTTP错误响应
type HTTPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// mapDomainErrorToHTTP 将领域错误映射为HTTP错误
func mapDomainErrorToHTTP(err error) (int, string) {
	if err == nil {
		return http.StatusOK, ""
	}

	switch {
	case errors.Is(err, application.ErrTaskNotFound):
		return http.StatusNotFound, "task not found"
	case errors.Is(err, domain.ErrInvalidStatusTransition):
		return http.StatusConflict, "invalid status transition"
	case errors.Is(err, domain.ErrTaskAlreadyStarted):
		return http.StatusConflict, "task already started"
	case errors.Is(err, domain.ErrTaskNotRunning):
		return http.StatusConflict, "task is not running"
	case errors.Is(err, domain.ErrTaskAlreadyFinished):
		return http.StatusConflict, "task already finished"
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	Name               string  `json:"name"`
	TaskRequirement    string  `json:"task_requirement"`    // 任务目标（必填）
	AcceptanceCriteria string  `json:"acceptance_criteria"` // 验收标准（必填）
	Description        string  `json:"description"`
	Type               string  `json:"type"`
	Timeout            int64   `json:"timeout"`
	MaxRetries         int     `json:"max_retries"`
	Priority           int     `json:"priority"`
	ParentID           *string `json:"parent_id"`
	TraceID            *string `json:"trace_id"`
	SpanID             *string `json:"span_id"`
}

// CreateTask 创建任务
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}

	if req.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "name is required"})
		return
	}

	if req.TaskRequirement == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "task_requirement is required"})
		return
	}

	if req.AcceptanceCriteria == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "acceptance_criteria is required"})
		return
	}

	if req.Type == "" {
		req.Type = domain.TaskTypeAgent.String()
	}
	if req.Type != domain.TaskTypeAgent.String() {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "only agent type is supported"})
		return
	}

	var parentID *domain.TaskID
	if req.ParentID != nil {
		id := domain.NewTaskID(*req.ParentID)
		parentID = &id
	}

	var traceID *domain.TraceID
	if req.TraceID != nil {
		tid := domain.NewTraceID(*req.TraceID)
		traceID = &tid
	}

	var spanID *domain.SpanID
	if req.SpanID != nil {
		sid := domain.NewSpanID(*req.SpanID)
		spanID = &sid
	}

	cmd := application.CreateTaskCommand{
		Name:               req.Name,
		TaskRequirement:    req.TaskRequirement,
		AcceptanceCriteria: req.AcceptanceCriteria,
		Description:        req.Description,
		Type:               domain.TaskTypeAgent,
		Timeout:            req.Timeout,
		MaxRetries:         req.MaxRetries,
		Priority:           req.Priority,
		ParentID:           parentID,
		TraceID:            traceID,
		SpanID:             spanID,
	}

	task, err := h.taskService.CreateTask(r.Context(), cmd)
	if err != nil {
		code, message := mapDomainErrorToHTTP(err)
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(HTTPError{Code: code, Message: message})
		return
	}

	if cmd.ParentID == nil {
		if err := h.taskService.StartTask(r.Context(), task.ID()); err != nil {
			code, message := mapDomainErrorToHTTP(err)
			w.WriteHeader(code)
			json.NewEncoder(w).Encode(HTTPError{Code: code, Message: message})
			return
		}
	}

	status := task.Status().String()
	if cmd.ParentID == nil {
		status = "running"
	}

	resp := map[string]interface{}{
		"id":         task.ID().String(),
		"trace_id":   task.TraceID().String(),
		"span_id":    task.SpanID().String(),
		"status":     status,
		"created_at": task.CreatedAt().UnixMilli(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// GetTask 获取任务
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.URL.Query().Get("id")
	if taskID == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	dto, err := h.queryService.GetTask(r.Context(), domain.NewTaskID(taskID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dto)
}

// ListAllTasks 获取所有任务
func (h *TaskHandler) ListAllTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.queryService.ListAllTasks(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// ListTasksByTrace 根据TraceID列任务
func (h *TaskHandler) ListTasksByTrace(w http.ResponseWriter, r *http.Request) {
	traceID := extractTraceID(r.URL.Path)
	if traceID == "" {
		http.Error(w, "trace_id is required", http.StatusBadRequest)
		return
	}

	tasks, err := h.queryService.ListTasksByTrace(r.Context(), domain.NewTraceID(traceID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// GetTaskTree 获取任务树
func (h *TaskHandler) GetTaskTree(w http.ResponseWriter, r *http.Request) {
	traceID := extractTraceID(r.URL.Path)
	if traceID == "" {
		http.Error(w, "trace_id is required", http.StatusBadRequest)
		return
	}

	tree, err := h.queryService.GetTaskTree(r.Context(), domain.NewTraceID(traceID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tree)
}

// StartTask 启动任务
func (h *TaskHandler) StartTask(w http.ResponseWriter, r *http.Request) {
	taskID := extractTaskID(r.URL.Path)
	if taskID == "" {
		http.Error(w, "task id is required", http.StatusBadRequest)
		return
	}

	if err := h.taskService.StartTask(r.Context(), domain.NewTaskID(taskID)); err != nil {
		code, message := mapDomainErrorToHTTP(err)
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(HTTPError{Code: code, Message: message})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "task started"})
}

// CancelTask 取消任务
func (h *TaskHandler) CancelTask(w http.ResponseWriter, r *http.Request) {
	taskID := extractTaskID(r.URL.Path)
	if taskID == "" {
		http.Error(w, "task id is required", http.StatusBadRequest)
		return
	}

	if err := h.taskService.CancelTask(r.Context(), domain.NewTaskID(taskID)); err != nil {
		code, message := mapDomainErrorToHTTP(err)
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(HTTPError{Code: code, Message: message})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "task cancelled"})
}

// ClearAllTasks 清空全部任务
func (h *TaskHandler) ClearAllTasks(w http.ResponseWriter, r *http.Request) {
	deleted, err := h.taskService.DeleteAllTasks(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "all tasks deleted",
		"deleted": deleted,
	})
}

// extractTraceID 从路径中提取 trace_id
func extractTraceID(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if (part == "traces" || part == "trace") && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// extractTaskID 从路径中提取 task id
func extractTaskID(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "tasks" && i+1 < len(parts) {
			id := parts[i+1]
			if id != "trace" {
				return id
			}
		}
	}
	return ""
}
