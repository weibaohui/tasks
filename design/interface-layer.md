# 接口层设计 (Interface Layer)

接口层负责接收外部请求，转换为应用层命令，返回响应。

## 目录

- [HTTP API设计](#http-api设计)
- [WebSocket设计](#websocket设计)

---

## HTTP API设计

```go
// interfaces/http/handler.go
package http

import (
    "encoding/json"
    "net/http"
    
    "github.com/example/taskmanager/application"
    "github.com/example/taskmanager/domain"
)

// TaskHandler HTTP处理器
type TaskHandler struct {
    taskService *application.TaskApplicationService
    queryService *application.QueryService
}

func NewTaskHandler(
    taskService *application.TaskApplicationService,
    queryService *application.QueryService,
) *TaskHandler {
    return &TaskHandler{
        taskService: taskService,
        queryService: queryService,
    }
}

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Type        string                 `json:"type"`
    Metadata    map[string]interface{} `json:"metadata"`
    Timeout     int64                  `json:"timeout"`    // 毫秒
    MaxRetries  int                    `json:"max_retries"`
    Priority    int                    `json:"priority"`
    ParentID    *string                `json:"parent_id"`
}

// CreateTaskResponse 创建任务响应
type CreateTaskResponse struct {
    ID        string `json:"id"`
    TraceID   string `json:"trace_id"`
    SpanID    string `json:"span_id"`
    Status    string `json:"status"`
    CreatedAt int64  `json:"created_at"`
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
    
    // 根据错误类型返回不同的HTTP状态码
    switch {
    case errors.Is(err, domain.ErrTaskNotFound):
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

// CreateTask 创建任务
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
    var req CreateTaskRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: err.Error()})
        return
    }
    
    // 参数校验
    if req.Name == "" {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(HTTPError{Code: http.StatusBadRequest, Message: "name is required"})
        return
    }
    
    // 转换请求为命令
    var parentID *domain.TaskID
    if req.ParentID != nil {
        id := domain.TaskID(*req.ParentID)
        parentID = &id
    }
    
    cmd := application.CreateTaskCommand{
        Name:        req.Name,
        Description: req.Description,
        Type:        domain.TaskType(req.Type),
        Metadata:    req.Metadata,
        Timeout:     req.Timeout,
        MaxRetries:  req.MaxRetries,
        Priority:    req.Priority,
        ParentID:    parentID,
    }
    
    task, err := h.taskService.CreateTask(r.Context(), cmd)
    if err != nil {
        code, message := mapDomainErrorToHTTP(err)
        w.WriteHeader(code)
        json.NewEncoder(w).Encode(HTTPError{Code: code, Message: message})
        return
    }
    
    resp := CreateTaskResponse{
        ID:        task.ID().String(),
        TraceID:   task.TraceID().String(),
        SpanID:    task.SpanID().String(),
        Status:    task.Status().String(),
        CreatedAt: task.CreatedAt().Unix(),
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(resp)
}

// GetTaskRequest 获取任务请求
type GetTaskRequest struct {
    ID string `json:"id"`
}

// GetTask 获取任务
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
    taskID := r.URL.Query().Get("id")
    if taskID == "" {
        http.Error(w, "id is required", http.StatusBadRequest)
        return
    }
    
    dto, err := h.queryService.GetTask(r.Context(), domain.TaskID(taskID))
    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(dto)
}

// ListTasksByTrace 根据TraceID列任务
func (h *TaskHandler) ListTasksByTrace(w http.ResponseWriter, r *http.Request) {
    traceID := r.URL.Query().Get("trace_id")
    if traceID == "" {
        http.Error(w, "trace_id is required", http.StatusBadRequest)
        return
    }
    
    tasks, err := h.queryService.ListTasksByTrace(r.Context(), domain.TraceID(traceID))
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(tasks)
}

// GetTaskTreeRequest 获取任务树请求
type GetTaskTreeRequest struct {
    TraceID string `json:"trace_id"`
}

// TaskTreeNodeResponse 任务树节点响应
type TaskTreeNodeResponse struct {
    Task     *application.GetTaskDTO   `json:"task"`
    Children []*TaskTreeNodeResponse   `json:"children"`
    Depth    int                       `json:"depth"`
}

// TaskTreeResponse 任务树响应
type TaskTreeResponse struct {
    Root     *TaskTreeNodeResponse `json:"root"`
    TraceID  string                `json:"trace_id"`
    Total    int                   `json:"total"`
    Complete int                   `json:"complete"`
}

// GetTaskTree 获取任务树
func (h *TaskHandler) GetTaskTree(w http.ResponseWriter, r *http.Request) {
    traceID := r.URL.Query().Get("trace_id")
    if traceID == "" {
        http.Error(w, "trace_id is required", http.StatusBadRequest)
        return
    }
    
    tree, err := h.taskService.GetTaskTree(r.Context(), domain.TraceID(traceID))
    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }
    
    resp := &TaskTreeResponse{
        TraceID:  tree.TraceID.String(),
        Total:    tree.Total,
        Complete: tree.Complete,
    }
    
    if tree.Root != nil {
        resp.Root = convertNodeToResponse(tree.Root)
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}

// convertNodeToResponse 转换节点为响应
func convertNodeToResponse(node *domain.TaskNode) *TaskTreeNodeResponse {
    // 转换领域节点为响应节点
    resp := &TaskTreeNodeResponse{
        Depth: node.Depth,
    }
    
    for _, child := range node.Children {
        resp.Children = append(resp.Children, convertNodeToResponse(child))
    }
    
    return resp
}

// CancelTaskRequest 取消任务请求
type CancelTaskRequest struct {
    ID     string `json:"id"`
    Reason string `json:"reason"`
}

// CancelTask 取消任务
func (h *TaskHandler) CancelTask(w http.ResponseWriter, r *http.Request) {
    var req CancelTaskRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    if err := h.taskService.CancelTask(r.Context(), domain.TaskID(req.ID), req.Reason); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}
```

### 路由定义

```go
// interfaces/http/router.go
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
            // GET /api/v1/tasks?trace_id=xxx - 获取任务列表
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
            handler.GetTaskTree(w, r)
        } else {
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        }
    })

    // POST /api/v1/tasks/{id}/cancel - 取消任务
    mux.HandleFunc("/api/v1/tasks/", func(w http.ResponseWriter, r *http.Request) {
        path := r.URL.Path
        if strings.HasSuffix(path, "/cancel") && r.Method == http.MethodPost {
            handler.CancelTask(w, r)
        } else {
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        }
    })

    return mux
}
```

---

## WebSocket设计

```go
// interfaces/ws/handler.go
package ws

import (
    "encoding/json"
    "net/http"
    
    "github.com/gorilla/websocket"
    
    "github.com/example/taskmanager/domain"
    "github.com/example/taskmanager/infrastructure/bus"
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}

// WebSocketHandler WebSocket处理器
type WebSocketHandler struct {
    eventBus *bus.EventBus
}

func NewWebSocketHandler(eventBus *bus.EventBus) *WebSocketHandler {
    return &WebSocketHandler{eventBus: eventBus}
}

// HandleWebSocket WebSocket连接处理
func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    defer conn.Close()
    
    // 获取订阅参数
    traceID := r.URL.Query().Get("trace_id")
    
    // 创建事件通道
    eventChan := make(chan domain.DomainEvent, 100)
    
    // 订阅事件
    unsubscribe := h.subscribeEvents(traceID, eventChan)
    defer unsubscribe()
    
    // 启动发送协程
    go func() {
        for event := range eventChan {
            msg, _ := json.Marshal(event)
            if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
                return
            }
        }
    }()
    
    // 保持连接
    for {
        _, _, err := conn.ReadMessage()
        if err != nil {
            break
        }
    }
}

// subscribeEvents 订阅事件
func (h *WebSocketHandler) subscribeEvents(
    traceID string,
    eventChan chan<- domain.DomainEvent,
) func() {
    var subscriptions []*bus.Subscription
    
    eventTypes := []string{
        "task:created",
        "task:started",
        "task:progress_updated",
        "task:completed",
        "task:failed",
        "task:cancelled",
    }
    
    for _, eventType := range eventTypes {
        et := eventType  // 闭包捕获
        sub := h.eventBus.Subscribe(et, func(event domain.DomainEvent) {
            // 如果指定了traceID，只发送匹配的事件
            if traceID != "" {
                // 从事件中获取traceID进行过滤
                // 实际应根据事件类型断言获取 TraceID
                // 这里简化处理
            }
            select {
            case eventChan <- event:
            default:
                // 通道满，丢弃事件
            }
        })
        subscriptions = append(subscriptions, sub)
    }
    
    // 返回取消订阅函数
    return func() {
        for _, sub := range subscriptions {
            h.eventBus.Unsubscribe(sub)
        }
    }
}
```

---

## API端点汇总

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/v1/tasks | 创建任务 |
| GET | /api/v1/tasks?id={id} | 获取任务详情 |
| GET | /api/v1/tasks/trace/{trace_id} | 获取任务列表（按 trace_id） |
| GET | /api/v1/traces/{trace_id}/tree | 获取任务树 |
| POST | /api/v1/tasks/{id}/cancel | 取消任务 |
| WS | /ws?trace_id={trace_id} | WebSocket事件流 |

---

## 请求响应示例

### 创建任务

**请求:**
```json
POST /api/v1/tasks
{
    "name": "数据处理任务",
    "description": "处理一批数据",
    "type": "data_processing",
    "timeout": 60000,
    "priority": 1
}
```

**响应:**
```json
{
    "id": "abc123...",
    "trace_id": "xyz789...",
    "span_id": "def456...",
    "status": "Pending",
    "created_at": 1700000000
}
```

### 获取任务树

**请求:**
```
GET /api/v1/tasks/tree?trace_id=xyz789...
```

**响应:**
```json
{
    "trace_id": "xyz789...",
    "total": 3,
    "complete": 1,
    "root": {
        "task": {
            "id": "abc123...",
            "name": "父任务",
            "status": "Running",
            "progress": {
                "percentage": 50.0
            }
        },
        "depth": 0,
        "children": [
            {
                "task": {
                    "id": "child1...",
                    "name": "子任务1",
                    "status": "Completed"
                },
                "depth": 1,
                "children": []
            }
        ]
    }
}
```
