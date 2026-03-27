/**
 * SQLite 任务仓储实现
 */
package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/weibh/taskmanager/domain"
	_ "modernc.org/sqlite"
)

// SQLiteTaskRepository SQLite任务仓储实现
type SQLiteTaskRepository struct {
	db *sql.DB
}

// NewSQLiteTaskRepository 创建SQLite任务仓储
func NewSQLiteTaskRepository(db *sql.DB) *SQLiteTaskRepository {
	return &SQLiteTaskRepository{db: db}
}

// Save 保存任务
func (r *SQLiteTaskRepository) Save(ctx context.Context, task *domain.Task) error {
	snap := task.ToSnapshot()

	metadata, _ := json.Marshal(snap.Metadata)
	progress, _ := json.Marshal(snap.Progress.ToMap())

	var resultJSON []byte
	if snap.Result != nil {
		resultJSON, _ = json.Marshal(snap.Result.ToMap())
	}

	var executionSummaryJSON []byte
	if snap.ExecutionSummary != nil {
		executionSummaryJSON, _ = json.Marshal(snap.ExecutionSummary)
	}

	query := `
		INSERT INTO tasks (id, trace_id, span_id, parent_id, name, description, type,
			acceptance_criteria, task_requirement, task_conclusion, user_code, agent_code, channel_code, session_key,
			execution_summary, todo_list, analysis, metadata, timeout, max_retries, priority, status, progress, result,
			error_msg, created_at, started_at, finished_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			acceptance_criteria=excluded.acceptance_criteria,
			task_requirement=excluded.task_requirement,
			task_conclusion=excluded.task_conclusion,
			user_code=excluded.user_code,
			agent_code=excluded.agent_code,
			channel_code=excluded.channel_code,
			session_key=excluded.session_key,
			execution_summary=excluded.execution_summary,
			todo_list=excluded.todo_list,
			analysis=excluded.analysis,
			status=excluded.status,
			progress=excluded.progress,
			result=excluded.result,
			error_msg=excluded.error_msg,
			metadata=excluded.metadata,
			started_at=excluded.started_at,
			finished_at=excluded.finished_at
	`

	var parentID interface{}
	if snap.ParentID != nil {
		parentID = snap.ParentID.String()
	}

	var startedAt, finishedAt interface{}
	if snap.StartedAt != nil {
		startedAt = snap.StartedAt.Unix()
	}
	if snap.FinishedAt != nil {
		finishedAt = snap.FinishedAt.Unix()
	}

	_, err := r.db.ExecContext(ctx, query,
		snap.ID.String(), snap.TraceID.String(), snap.SpanID.String(), parentID,
		snap.Name, snap.Description, snap.Type.String(),
		snap.AcceptanceCriteria, snap.TaskRequirement, snap.TaskConclusion,
		snap.UserCode, snap.AgentCode, snap.ChannelCode, snap.SessionKey,
		executionSummaryJSON, snap.TodoList, snap.Analysis, metadata,
		snap.Timeout.Milliseconds(), snap.MaxRetries, snap.Priority, int(snap.Status),
		progress, resultJSON, snap.ErrorMsg, snap.CreatedAt.Unix(),
		startedAt, finishedAt,
	)

	return err
}

// FindByID 根据ID查找任务
func (r *SQLiteTaskRepository) FindByID(ctx context.Context, id domain.TaskID) (*domain.Task, error) {
	query := `
		SELECT id, trace_id, span_id, parent_id, name, description, type,
			   acceptance_criteria, task_requirement, task_conclusion, user_code, agent_code, channel_code, session_key,
			   execution_summary, todo_list, analysis, metadata, timeout, max_retries, priority, status, progress, result,
			   error_msg, created_at, started_at, finished_at
		FROM tasks WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id.String())
	return r.scanToTask(row)
}

// FindAll 获取所有任务
func (r *SQLiteTaskRepository) FindAll(ctx context.Context) ([]*domain.Task, error) {
	query := `
		SELECT id, trace_id, span_id, parent_id, name, description, type,
			   acceptance_criteria, task_requirement, task_conclusion, user_code, agent_code, channel_code, session_key,
			   execution_summary, todo_list, analysis, metadata, timeout, max_retries, priority, status, progress, result,
			   error_msg, created_at, started_at, finished_at
		FROM tasks ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanToTasks(rows)
}

// FindByTraceID 根据TraceID查找所有任务
func (r *SQLiteTaskRepository) FindByTraceID(ctx context.Context, traceID domain.TraceID) ([]*domain.Task, error) {
	query := `
		SELECT id, trace_id, span_id, parent_id, name, description, type,
			   acceptance_criteria, task_requirement, task_conclusion, user_code, agent_code, channel_code, session_key,
			   execution_summary, todo_list, analysis, metadata, timeout, max_retries, priority, status, progress, result,
			   error_msg, created_at, started_at, finished_at
		FROM tasks WHERE trace_id = ? ORDER BY created_at`

	rows, err := r.db.QueryContext(ctx, query, traceID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanToTasks(rows)
}

// FindByParentID 根据父任务ID查找子任务
func (r *SQLiteTaskRepository) FindByParentID(ctx context.Context, parentID domain.TaskID) ([]*domain.Task, error) {
	query := `
		SELECT id, trace_id, span_id, parent_id, name, description, type,
			   acceptance_criteria, task_requirement, task_conclusion, user_code, agent_code, channel_code, session_key,
			   execution_summary, todo_list, analysis, metadata, timeout, max_retries, priority, status, progress, result,
			   error_msg, created_at, started_at, finished_at
		FROM tasks WHERE parent_id = ?`

	rows, err := r.db.QueryContext(ctx, query, parentID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanToTasks(rows)
}

// FindByStatus 根据状态查找任务
func (r *SQLiteTaskRepository) FindByStatus(ctx context.Context, status domain.TaskStatus) ([]*domain.Task, error) {
	query := `
		SELECT id, trace_id, span_id, parent_id, name, description, type,
			   acceptance_criteria, task_requirement, task_conclusion, user_code, agent_code, channel_code, session_key,
			   execution_summary, todo_list, analysis, metadata, timeout, max_retries, priority, status, progress, result,
			   error_msg, created_at, started_at, finished_at
		FROM tasks WHERE status = ?`

	rows, err := r.db.QueryContext(ctx, query, int(status))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanToTasks(rows)
}

// FindRunningTasks 查找所有运行中的任务
func (r *SQLiteTaskRepository) FindRunningTasks(ctx context.Context) ([]*domain.Task, error) {
	return r.FindByStatus(ctx, domain.TaskStatusRunning)
}

// Delete 删除任务
func (r *SQLiteTaskRepository) Delete(ctx context.Context, id domain.TaskID) error {
	query := `DELETE FROM tasks WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id.String())
	return err
}

// Exists 判断任务是否存在
func (r *SQLiteTaskRepository) Exists(ctx context.Context, id domain.TaskID) (bool, error) {
	query := `SELECT 1 FROM tasks WHERE id = ? LIMIT 1`
	var n int
	err := r.db.QueryRowContext(ctx, query, id.String()).Scan(&n)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// scanToTask 将 row 扫描为 Task
func (r *SQLiteTaskRepository) scanToTask(row *sql.Row) (*domain.Task, error) {
	var snap domain.TaskSnapshot
	var metadataJSON, progressJSON, resultJSON, executionSummaryJSON []byte
	var idStr, traceIDStr, spanIDStr string
	var parentIDStr *string
	var typeStr string
	var statusInt int
	var createdAtUnix int64
	var startedAtUnix, finishedAtUnix *int64
	var timeoutMs int64
	var acceptanceCriteria, taskRequirement, taskConclusion, userCode, agentCode, channelCode, sessionKey sql.NullString
	var todoList, analysis sql.NullString

	err := row.Scan(
		&idStr, &traceIDStr, &spanIDStr, &parentIDStr,
		&snap.Name, &snap.Description, &typeStr,
		&acceptanceCriteria, &taskRequirement, &taskConclusion,
		&userCode, &agentCode, &channelCode, &sessionKey,
		&executionSummaryJSON, &todoList, &analysis, &metadataJSON,
		&timeoutMs, &snap.MaxRetries, &snap.Priority, &statusInt,
		&progressJSON, &resultJSON, &snap.ErrorMsg, &createdAtUnix,
		&startedAtUnix, &finishedAtUnix,
	)
	if err != nil {
		return nil, err
	}

	snap.ID = domain.NewTaskID(idStr)
	snap.TraceID = domain.NewTraceID(traceIDStr)
	snap.SpanID = domain.NewSpanID(spanIDStr)
	snap.AcceptanceCriteria = acceptanceCriteria.String
	snap.TaskRequirement = taskRequirement.String
	snap.TaskConclusion = taskConclusion.String
	snap.UserCode = userCode.String
	snap.AgentCode = agentCode.String
	snap.ChannelCode = channelCode.String
	snap.SessionKey = sessionKey.String
	snap.TodoList = todoList.String
	snap.Analysis = analysis.String

	// 反序列化
	if err := json.Unmarshal(metadataJSON, &snap.Metadata); err != nil {
		snap.Metadata = make(map[string]interface{})
	}

	if err := json.Unmarshal(executionSummaryJSON, &snap.ExecutionSummary); err != nil {
		snap.ExecutionSummary = nil
	}

	var progressMap map[string]interface{}
	if err := json.Unmarshal(progressJSON, &progressMap); err == nil {
		snap.Progress = mapToProgress(progressMap)
	}

	if resultJSON != nil {
		var resultMap map[string]interface{}
		if err := json.Unmarshal(resultJSON, &resultMap); err == nil {
			snap.Result = mapToResult(resultMap)
		}
	}

	snap.Type, _ = domain.ParseTaskType(typeStr)
	snap.Status = domain.TaskStatus(statusInt)
	snap.Timeout = time.Duration(timeoutMs) * time.Millisecond
	snap.CreatedAt = time.Unix(createdAtUnix, 0)

	if parentIDStr != nil {
		id := domain.NewTaskID(*parentIDStr)
		snap.ParentID = &id
	}

	if startedAtUnix != nil {
		t := time.Unix(*startedAtUnix, 0)
		snap.StartedAt = &t
	}

	if finishedAtUnix != nil {
		t := time.Unix(*finishedAtUnix, 0)
		snap.FinishedAt = &t
	}

	task := &domain.Task{}
	task.FromSnapshot(&snap)
	return task, nil
}

// scanToTasks 扫描多个 task
func (r *SQLiteTaskRepository) scanToTasks(rows *sql.Rows) ([]*domain.Task, error) {
	var tasks []*domain.Task
	for rows.Next() {
		var snap domain.TaskSnapshot
		var metadataJSON, progressJSON, resultJSON, executionSummaryJSON []byte
		var idStr, traceIDStr, spanIDStr string
		var parentIDStr *string
		var typeStr string
		var statusInt int
		var createdAtUnix int64
		var startedAtUnix, finishedAtUnix *int64
		var timeoutMs int64
		var acceptanceCriteria, taskRequirement, taskConclusion, userCode, agentCode, channelCode, sessionKey sql.NullString
		var todoList, analysis sql.NullString

		err := rows.Scan(
			&idStr, &traceIDStr, &spanIDStr, &parentIDStr,
			&snap.Name, &snap.Description, &typeStr,
			&acceptanceCriteria, &taskRequirement, &taskConclusion,
			&userCode, &agentCode, &channelCode, &sessionKey,
			&executionSummaryJSON, &todoList, &analysis, &metadataJSON,
			&timeoutMs, &snap.MaxRetries, &snap.Priority, &statusInt,
			&progressJSON, &resultJSON, &snap.ErrorMsg, &createdAtUnix,
			&startedAtUnix, &finishedAtUnix,
		)
		if err != nil {
			return nil, err
		}

		snap.ID = domain.NewTaskID(idStr)
		snap.TraceID = domain.NewTraceID(traceIDStr)
		snap.SpanID = domain.NewSpanID(spanIDStr)
		snap.AcceptanceCriteria = acceptanceCriteria.String
		snap.TaskRequirement = taskRequirement.String
		snap.TaskConclusion = taskConclusion.String
		snap.UserCode = userCode.String
		snap.AgentCode = agentCode.String
		snap.ChannelCode = channelCode.String
		snap.SessionKey = sessionKey.String
		snap.TodoList = todoList.String
		snap.Analysis = analysis.String

		// 反序列化
		if err := json.Unmarshal(metadataJSON, &snap.Metadata); err != nil {
			snap.Metadata = make(map[string]interface{})
		}

		if err := json.Unmarshal(executionSummaryJSON, &snap.ExecutionSummary); err != nil {
			snap.ExecutionSummary = nil
		}

		var progressMap map[string]interface{}
		if err := json.Unmarshal(progressJSON, &progressMap); err == nil {
			snap.Progress = mapToProgress(progressMap)
		}

		if resultJSON != nil {
			var resultMap map[string]interface{}
			if err := json.Unmarshal(resultJSON, &resultMap); err == nil {
				snap.Result = mapToResult(resultMap)
			}
		}

		snap.Type, _ = domain.ParseTaskType(typeStr)
		snap.Status = domain.TaskStatus(statusInt)
		snap.Timeout = time.Duration(timeoutMs) * time.Millisecond
		snap.CreatedAt = time.Unix(createdAtUnix, 0)

		if parentIDStr != nil {
			id := domain.NewTaskID(*parentIDStr)
			snap.ParentID = &id
		}

		if startedAtUnix != nil {
			t := time.Unix(*startedAtUnix, 0)
			snap.StartedAt = &t
		}

		if finishedAtUnix != nil {
			t := time.Unix(*finishedAtUnix, 0)
			snap.FinishedAt = &t
		}

		task := &domain.Task{}
		task.FromSnapshot(&snap)
		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

// mapToProgress 将 map 转换为 Progress
func mapToProgress(m map[string]interface{}) domain.Progress {
	p := domain.NewProgress()

	if total, ok := m["total"].(float64); ok {
		if current, ok := m["current"].(float64); ok {
			stage, _ := m["stage"].(string)
			detail, _ := m["detail"].(string)
			p.Update(int(total), int(current), stage, detail)
		}
	}

	return p
}

// mapToResult 将 map 转换为 Result
func mapToResult(m map[string]interface{}) *domain.Result {
	data := m["data"]
	message, _ := m["message"].(string)
	r := domain.NewResult(data, message)
	return &r
}
