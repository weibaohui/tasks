/**
 * 应用层 DTO 定义
 */
package application

// GetTaskDTO 获取任务详情 DTO
type GetTaskDTO struct {
	ID                   string      `json:"id"`
	TraceID              string      `json:"trace_id"`
	SpanID               string      `json:"span_id"`
	ParentID             *string     `json:"parent_id,omitempty"`
	Name                 string      `json:"name"`
	Description          string      `json:"description"`
	Type                 string      `json:"type"`
	Status               string      `json:"status"`
	Progress             ProgressDTO `json:"progress"`
	Result               *ResultDTO  `json:"result,omitempty"`
	Error                string      `json:"error,omitempty"`
	Depth                int         `json:"depth"`
	ParentSpan           string      `json:"parent_span"`
	Timeout              int64       `json:"timeout"`
	MaxRetries           int         `json:"max_retries"`
	Priority             int         `json:"priority"`
	CreatedAt            int64       `json:"created_at"`
	StartedAt            *int64      `json:"started_at,omitempty"`
	FinishedAt           *int64      `json:"finished_at,omitempty"`
	AcceptanceCriteria   string      `json:"acceptance_criteria"`
	TaskRequirement      string      `json:"task_requirement"`
	TaskConclusion       string      `json:"task_conclusion"`
	UserCode             string      `json:"user_code"`
	AgentCode            string      `json:"agent_code"`
	ChannelCode          string      `json:"channel_code"`
	SessionKey           string      `json:"session_key"`
	TodoList             string      `json:"todo_list"`
	Analysis             string      `json:"analysis"`
}

// ProgressDTO 进度 DTO
type ProgressDTO struct {
	Value     int   `json:"value"`
	UpdatedAt int64 `json:"updated_at"`
}

// ResultDTO 结果 DTO
type ResultDTO struct {
	Data           interface{} `json:"data"`
	Message        string      `json:"message"`
	TaskConclusion string     `json:"task_conclusion,omitempty"`
}

// ListTasksDTO 任务列表 DTO
type ListTasksDTO struct {
	Tasks []*GetTaskDTO `json:"tasks"`
	Total int           `json:"total"`
}

// TaskTreeNodeDTO 任务树节点 DTO
type TaskTreeNodeDTO struct {
	Task     *GetTaskDTO        `json:"task"`
	Children []*TaskTreeNodeDTO `json:"children"`
}
