package persistence

import (
	"database/sql"
	"fmt"
	"time"
)

// MigrateClaudeRuntimeColumns 兼容旧数据库：将 claude_runtime_* 列重命名为 agent_runtime_*
// 并在 requirements 表中新增 agent_runtime_agent_type 列
func MigrateClaudeRuntimeColumns(db *sql.DB) error {
	columns, err := getTableColumns(db, "requirements")
	if err != nil {
		return fmt.Errorf("获取 requirements 表列信息失败: %w", err)
	}

	// 旧列名 -> 新列名映射
	renames := map[string]string{
		"claude_runtime_status":      "agent_runtime_status",
		"claude_runtime_started_at":  "agent_runtime_started_at",
		"claude_runtime_ended_at":    "agent_runtime_ended_at",
		"claude_runtime_error":       "agent_runtime_error",
		"claude_runtime_result":      "agent_runtime_result",
		"claude_runtime_prompt":      "agent_runtime_prompt",
	}

	for oldName, newName := range renames {
		if _, exists := columns[oldName]; exists {
			if _, err := db.Exec(fmt.Sprintf("ALTER TABLE requirements RENAME COLUMN %s TO %s", oldName, newName)); err != nil {
				return fmt.Errorf("重命名列 %s -> %s 失败: %w", oldName, newName, err)
			}
		}
	}

	// 确保 agent_runtime_agent_type 列存在
	if _, exists := columns["agent_runtime_agent_type"]; !exists {
		if _, err := db.Exec("ALTER TABLE requirements ADD COLUMN agent_runtime_agent_type TEXT"); err != nil {
			return fmt.Errorf("添加 agent_runtime_agent_type 列失败: %w", err)
		}
	}

	return nil
}

// MigrateProgressDataColumn 兼容旧数据库：在 requirements 表中添加 progress_data 列
func MigrateProgressDataColumn(db *sql.DB) error {
	columns, err := getTableColumns(db, "requirements")
	if err != nil {
		return fmt.Errorf("获取 requirements 表列信息失败: %w", err)
	}

	if _, exists := columns["progress_data"]; !exists {
		if _, err := db.Exec("ALTER TABLE requirements ADD COLUMN progress_data TEXT"); err != nil {
			return fmt.Errorf("添加 progress_data 列失败: %w", err)
		}
	}

	return nil
}

// MigrateRequirementTypeSystemColumn 兼容旧数据库：在 requirement_types 表中添加 is_system 列
func MigrateRequirementTypeSystemColumn(db *sql.DB) error {
	columns, err := getTableColumns(db, "requirement_types")
	if err != nil {
		return fmt.Errorf("获取 requirement_types 表列信息失败: %w", err)
	}

	if _, exists := columns["is_system"]; !exists {
		if _, err := db.Exec("ALTER TABLE requirement_types ADD COLUMN is_system INTEGER NOT NULL DEFAULT 0"); err != nil {
			return fmt.Errorf("添加 is_system 列失败: %w", err)
		}
	}

	// 兼容旧数据：将已有的 normal 和 heartbeat 标记为系统类型
	if _, err := db.Exec("UPDATE requirement_types SET is_system = 1 WHERE code IN ('normal', 'heartbeat')"); err != nil {
		return fmt.Errorf("更新系统类型标志失败: %w", err)
	}

	return nil
}

// MigrateMaxConcurrentAgentsColumn 兼容旧数据库：在 projects 表中添加 max_concurrent_agents 列
func MigrateMaxConcurrentAgentsColumn(db *sql.DB) error {
	columns, err := getTableColumns(db, "projects")
	if err != nil {
		return fmt.Errorf("获取 projects 表列信息失败: %w", err)
	}

	if _, exists := columns["max_concurrent_agents"]; !exists {
		if _, err := db.Exec("ALTER TABLE projects ADD COLUMN max_concurrent_agents INTEGER NOT NULL DEFAULT 2"); err != nil {
			return fmt.Errorf("添加 max_concurrent_agents 列失败: %w", err)
		}
	}

	return nil
}

// MigrateRequirementAgentInfoColumns 兼容旧数据库：在 requirements 表中添加 agent 名称和分身来源列
func MigrateRequirementAgentInfoColumns(db *sql.DB) error {
	columns, err := getTableColumns(db, "requirements")
	if err != nil {
		return fmt.Errorf("获取 requirements 表列信息失败: %w", err)
	}

	columnMigrations := map[string]string{
		"assignee_agent_name":       "ALTER TABLE requirements ADD COLUMN assignee_agent_name TEXT",
		"replica_agent_name":        "ALTER TABLE requirements ADD COLUMN replica_agent_name TEXT",
		"replica_agent_shadow_from": "ALTER TABLE requirements ADD COLUMN replica_agent_shadow_from TEXT",
	}

	for columnName, sqlStmt := range columnMigrations {
		if _, exists := columns[columnName]; !exists {
			if _, err := db.Exec(sqlStmt); err != nil {
				return fmt.Errorf("添加 %s 列失败: %w", columnName, err)
			}
		}
	}

	return nil
}

// MigrateHeartbeatToTable 将旧项目的心跳配置迁移到独立的 heartbeats 表
func MigrateHeartbeatToTable(db *sql.DB) error {
	// 1. 创建 heartbeats 表（若不存在）
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS heartbeats (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		name TEXT NOT NULL,
		enabled INTEGER NOT NULL DEFAULT 1,
		interval_minutes INTEGER NOT NULL DEFAULT 60,
		md_content TEXT NOT NULL DEFAULT '',
		agent_code TEXT NOT NULL DEFAULT '',
		requirement_type TEXT NOT NULL DEFAULT 'heartbeat',
		sort_order INTEGER NOT NULL DEFAULT 0,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_heartbeats_project_id ON heartbeats(project_id);
	CREATE INDEX IF NOT EXISTS idx_heartbeats_enabled ON heartbeats(enabled);
	`
	if _, err := db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("创建 heartbeats 表失败: %w", err)
	}

	// 2. 检查是否需要迁移：projects 表是否还有旧的心跳列
	columns, err := getTableColumns(db, "projects")
	if err != nil {
		return fmt.Errorf("获取 projects 表列信息失败: %w", err)
	}
	if _, hasHeartbeatEnabled := columns["heartbeat_enabled"]; !hasHeartbeatEnabled {
		// 没有旧列，说明不需要迁移
		return nil
	}

	// 3. 查询所有启用心跳且配置了 agent_code 的旧项目
	rows, err := db.Query(`
		SELECT id, heartbeat_interval_minutes, heartbeat_md_content, agent_code
		FROM projects
		WHERE heartbeat_enabled = 1 AND agent_code != ''
	`)
	if err != nil {
		return fmt.Errorf("查询旧心跳项目失败: %w", err)
	}

	type oldProject struct {
		projectID       string
		intervalMinutes int
		mdContent       string
		agentCode       string
	}
	var projects []oldProject
	for rows.Next() {
		var p oldProject
		if err := rows.Scan(&p.projectID, &p.intervalMinutes, &p.mdContent, &p.agentCode); err != nil {
			rows.Close()
			return fmt.Errorf("扫描旧心跳项目失败: %w", err)
		}
		projects = append(projects, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("读取旧心跳项目失败: %w", err)
	}

	now := time.Now().Unix()
	for _, p := range projects {
		heartbeatID := "hb_" + p.projectID
		// 检查是否已存在
		var exists int
		if err := db.QueryRow(`SELECT 1 FROM heartbeats WHERE id = ?`, heartbeatID).Scan(&exists); err == nil {
			continue // 已存在，跳过
		}

		_, err := db.Exec(`
			INSERT INTO heartbeats (id, project_id, name, enabled, interval_minutes, md_content, agent_code, requirement_type, sort_order, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, heartbeatID, p.projectID, "默认心跳", 1, p.intervalMinutes, p.mdContent, p.agentCode, "heartbeat", 0, now, now)
		if err != nil {
			return fmt.Errorf("插入默认心跳失败 project=%s: %w", p.projectID, err)
		}
	}
	return nil
}

func getTableColumns(db *sql.DB, tableName string) (map[string]bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return nil, err
		}
		columns[name] = true
	}
	return columns, rows.Err()
}
