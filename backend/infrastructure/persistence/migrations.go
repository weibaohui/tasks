package persistence

import (
	"database/sql"
	"fmt"
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
