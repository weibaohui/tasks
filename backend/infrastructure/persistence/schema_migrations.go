/**
 * 数据库 Schema 定义 - 迁移
 */
package persistence

import (
	"database/sql"
	"fmt"
)

// migrateTasksNewColumns 迁移 tasks 表新增字段
func migrateTasksNewColumns(db *sql.DB) error {
	newColumns := []struct {
		name       string
		sqlType    string
		oldName    string // 如果旧列存在，数据迁移到新列
		defaultVal string // 默认值
	}{
		{"acceptance_criteria", "TEXT", "", ""},
		{"task_requirement", "TEXT", "llm_reason", ""},
		{"task_conclusion", "TEXT", "", ""},
		{"user_code", "TEXT", "", ""},
		{"agent_code", "TEXT", "", ""},
		{"channel_code", "TEXT", "", ""},
		{"session_key", "TEXT", "", ""},
		{"todo_list", "TEXT", "", ""},
		{"analysis", "TEXT", "", ""},
		{"depth", "INTEGER", "", "0"},
		{"parent_span", "TEXT", "", ""},
		{"subtask_records", "TEXT", "", ""},
	}

	for _, col := range newColumns {
		has, err := tableHasColumn(db, "tasks", col.name)
		if err != nil {
			return err
		}
		if !has {
			if col.oldName != "" {
				// 检查旧列是否存在，如果存在则重命名
				oldHas, err := tableHasColumn(db, "tasks", col.oldName)
				if err != nil {
					return err
				}
				if oldHas {
					// 重命名旧列为新列
					if _, err := db.Exec(fmt.Sprintf("ALTER TABLE tasks RENAME COLUMN %s TO %s", col.oldName, col.name)); err != nil {
						return err
					}
					continue
				}
			}
			// 添加新列
			sql := fmt.Sprintf("ALTER TABLE tasks ADD COLUMN %s %s", col.name, col.sqlType)
			if col.defaultVal != "" {
				sql += " NOT NULL DEFAULT " + col.defaultVal
			}
			if _, err := db.Exec(sql); err != nil {
				return err
			}
		}
	}

	return nil
}

// migrateAgentTypeColumn 迁移 agents 表新增 agent_type 字段
func migrateAgentTypeColumn(db *sql.DB) error {
	has, err := tableHasColumn(db, "agents", "agent_type")
	if err != nil {
		return err
	}
	if !has {
		if _, err := db.Exec("ALTER TABLE agents ADD COLUMN agent_type TEXT NOT NULL DEFAULT 'BareLLM'"); err != nil {
			return err
		}
	}
	return nil
}

// migrateDropResultColumn 删除 tasks 表的 result 列
func migrateDropResultColumn(db *sql.DB) error {
	has, err := tableHasColumn(db, "tasks", "result")
	if err != nil {
		return err
	}
	if has {
		if _, err := db.Exec("ALTER TABLE tasks DROP COLUMN result"); err != nil {
			return err
		}
	}
	return nil
}

// migrateTasksTimeoutToSeconds 将 tasks 表中 timeout 从毫秒转换为秒
// 通过检测 timeout > 1000 来判断是否为毫秒值（旧数据），并进行转换
func migrateTasksTimeoutToSeconds(db *sql.DB) error {
	// 检查是否已执行过迁移（通过检测是否有 timeout > 1000 且 < 1e10 的记录）
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE timeout > 1000 AND timeout < 10000000000`).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		// 没有需要转换的数据，可能是新数据或已转换
		return nil
	}
	// 转换：毫秒值 / 1000 = 秒值
	_, err = db.Exec(`UPDATE tasks SET timeout = timeout / 1000 WHERE timeout > 1000 AND timeout < 10000000000`)
	return err
}

func migrateAgentMCPBindingColumn(db *sql.DB) error {
	hasOld, err := tableHasColumn(db, "agent_mcp_bindings", "is_enabled")
	if err != nil {
		return err
	}
	hasNew, err := tableHasColumn(db, "agent_mcp_bindings", "is_active")
	if err != nil {
		return err
	}
	if hasOld && !hasNew {
		if _, err := db.Exec("ALTER TABLE agent_mcp_bindings RENAME COLUMN is_enabled TO is_active"); err != nil {
			return err
		}
	}
	return nil
}

func migrateRequirementsNewColumns(db *sql.DB) error {
	has, err := tableHasColumn(db, "requirements", "temp_workspace_root")
	if err != nil {
		return err
	}
	if !has {
		if _, err := db.Exec("ALTER TABLE requirements ADD COLUMN temp_workspace_root TEXT"); err != nil {
			return err
		}
	}
	hasDispatchSessionKey, err := tableHasColumn(db, "requirements", "dispatch_session_key")
	if err != nil {
		return err
	}
	if !hasDispatchSessionKey {
		if _, err := db.Exec("ALTER TABLE requirements ADD COLUMN dispatch_session_key TEXT"); err != nil {
			return err
		}
	}

	// 重命名 assignee_agent_id -> assignee_agent_code
	hasOldAssignee, err := tableHasColumn(db, "requirements", "assignee_agent_id")
	if err != nil {
		return err
	}
	hasNewAssignee, err := tableHasColumn(db, "requirements", "assignee_agent_code")
	if err != nil {
		return err
	}
	if hasOldAssignee && !hasNewAssignee {
		if _, err := db.Exec("ALTER TABLE requirements RENAME COLUMN assignee_agent_id TO assignee_agent_code"); err != nil {
			return err
		}
	}

	// 重命名 replica_agent_id -> replica_agent_code
	hasOldReplica, err := tableHasColumn(db, "requirements", "replica_agent_id")
	if err != nil {
		return err
	}
	hasNewReplica, err := tableHasColumn(db, "requirements", "replica_agent_code")
	if err != nil {
		return err
	}
	if hasOldReplica && !hasNewReplica {
		if _, err := db.Exec("ALTER TABLE requirements RENAME COLUMN replica_agent_id TO replica_agent_code"); err != nil {
			return err
		}
	}

	return nil
}

// migrateRequirementsClaudeRuntime 迁移 requirements 表新增 claude_runtime 相关字段
func migrateRequirementsClaudeRuntime(db *sql.DB) error {
	columns := []struct {
		name         string
		sqlType      string
		backfillZero bool
	}{
		{"claude_runtime_status", "TEXT", false},
		{"claude_runtime_started_at", "INTEGER", false},
		{"claude_runtime_ended_at", "INTEGER", false},
		{"claude_runtime_error", "TEXT", false},
		{"claude_runtime_result", "TEXT", false},
		{"claude_runtime_prompt", "TEXT", false},
		{"trace_id", "TEXT", false},
		{"prompt_tokens", "INTEGER NOT NULL DEFAULT 0", true},
		{"completion_tokens", "INTEGER NOT NULL DEFAULT 0", true},
		{"total_tokens", "INTEGER NOT NULL DEFAULT 0", true},
	}

	for _, col := range columns {
		has, err := tableHasColumn(db, "requirements", col.name)
		if err != nil {
			return err
		}
		if !has {
			if _, err := db.Exec(fmt.Sprintf("ALTER TABLE requirements ADD COLUMN %s %s", col.name, col.sqlType)); err != nil {
				return err
			}
		}
		// 对需要回填的列，将历史 NULL 值更新为 0
		if col.backfillZero {
			if _, err := db.Exec(fmt.Sprintf("UPDATE requirements SET %s = 0 WHERE %s IS NULL", col.name, col.name)); err != nil {
				return err
			}
		}
	}
	return nil
}

// migrateRequirementType 迁移 requirements 表新增 requirement_type 字段
func migrateRequirementType(db *sql.DB) error {
	has, err := tableHasColumn(db, "requirements", "requirement_type")
	if err != nil {
		return err
	}
	if !has {
		if _, err := db.Exec("ALTER TABLE requirements ADD COLUMN requirement_type TEXT NOT NULL DEFAULT 'normal'"); err != nil {
			return err
		}
	}
	return nil
}

func migrateLLMProviderTypeColumn(db *sql.DB) error {
	hasColumn, err := tableHasColumn(db, "llm_providers", "provider_type")
	if err != nil {
		return err
	}
	if !hasColumn {
		if _, err := db.Exec("ALTER TABLE llm_providers ADD COLUMN provider_type TEXT NOT NULL DEFAULT 'openai'"); err != nil {
			return err
		}
	}
	return nil
}

// migrateAgentShadowFrom 迁移 agents 表新增 shadow_from 字段
func migrateAgentShadowFrom(db *sql.DB) error {
	has, err := tableHasColumn(db, "agents", "shadow_from")
	if err != nil {
		return err
	}
	if !has {
		if _, err := db.Exec("ALTER TABLE agents ADD COLUMN shadow_from TEXT"); err != nil {
			return err
		}
	}
	return nil
}

// migrateProjectsNewColumns 迁移 projects 表新增字段
func migrateProjectsNewColumns(db *sql.DB) error {
	// 添加 dispatch_channel_code 列
	has, err := tableHasColumn(db, "projects", "dispatch_channel_code")
	if err != nil {
		return err
	}
	if !has {
		if _, err := db.Exec("ALTER TABLE projects ADD COLUMN dispatch_channel_code TEXT NOT NULL DEFAULT ''"); err != nil {
			return err
		}
	}

	// 添加 dispatch_session_key 列
	has, err = tableHasColumn(db, "projects", "dispatch_session_key")
	if err != nil {
		return err
	}
	if !has {
		if _, err := db.Exec("ALTER TABLE projects ADD COLUMN dispatch_session_key TEXT NOT NULL DEFAULT ''"); err != nil {
			return err
		}
	}

	// 将 heartbeat_agent_id 重命名为 agent_code
	hasOldColumn, err := tableHasColumn(db, "projects", "heartbeat_agent_id")
	if err != nil {
		return err
	}
	hasNewColumn, err := tableHasColumn(db, "projects", "agent_code")
	if err != nil {
		return err
	}
	if hasOldColumn && !hasNewColumn {
		if _, err := db.Exec("ALTER TABLE projects RENAME COLUMN heartbeat_agent_id TO agent_code"); err != nil {
			return err
		}
	}

	return nil
}

// migrateConversationRecordsTimestampToMillis 将 conversation_records 表中的秒级时间戳转换为毫秒级
// 通过判断时间戳数值大小来区分：秒级时间戳约10位，毫秒级约13位
func migrateConversationRecordsTimestampToMillis(db *sql.DB) error {
	// 检查 timestamp 列的数据范围
	// 如果最大值小于 1e12（约 2001-09-09，秒级），说明是秒级时间戳
	var maxTimestamp sql.NullInt64
	var maxCreatedAt sql.NullInt64

	err := db.QueryRow(`SELECT MAX(timestamp), MAX(created_at) FROM conversation_records`).Scan(&maxTimestamp, &maxCreatedAt)
	if err != nil {
		return err
	}

	// 如果没有数据，直接返回
	if !maxTimestamp.Valid && !maxCreatedAt.Valid {
		return nil
	}

	// 判断是否为秒级时间戳（小于 1e12 认为是秒级）
	isSeconds := false
	if maxTimestamp.Valid && maxTimestamp.Int64 > 0 && maxTimestamp.Int64 < 1e12 {
		isSeconds = true
	}
	if maxCreatedAt.Valid && maxCreatedAt.Int64 > 0 && maxCreatedAt.Int64 < 1e12 {
		isSeconds = true
	}

	if !isSeconds {
		// 已经是毫秒级或没有需要转换的数据
		return nil
	}

	// 执行转换：秒级 * 1000 = 毫秒级
	_, err = db.Exec(`UPDATE conversation_records SET timestamp = timestamp * 1000, created_at = created_at * 1000 WHERE timestamp < 1000000000000`)
	return err
}

// migrateAgentLLMProviderID 迁移 agents 表新增 llm_provider_id 字段
func migrateAgentLLMProviderID(db *sql.DB) error {
	has, err := tableHasColumn(db, "agents", "llm_provider_id")
	if err != nil {
		return err
	}
	if !has {
		if _, err := db.Exec("ALTER TABLE agents ADD COLUMN llm_provider_id TEXT REFERENCES llm_providers(id)"); err != nil {
			return err
		}
	}
	return nil
}

// migrateRequirementsTraceIDIndex 迁移 requirements 表新增 trace_id 索引
func migrateRequirementsTraceIDIndex(db *sql.DB) error {
	_, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_requirements_trace_id ON requirements(trace_id)")
	return err
}

// migrateUserTokenValue 迁移 user_tokens 表新增 token_value 列
func migrateUserTokenValue(db *sql.DB) error {
	exists, err := tableHasColumn(db, "user_tokens", "token_value")
	if err != nil {
		return err
	}
	if !exists {
		if _, err := db.Exec("ALTER TABLE user_tokens ADD COLUMN token_value TEXT"); err != nil {
			return err
		}
	}
	return nil
}
