/**
 * 数据库 Schema 定义
 * 注意：此文件已拆分为多个文件
 * - schema_tables.go: 表结构定义
 * - schema_indexes.go: 索引定义
 * - schema_migrations.go: 数据库迁移函数
 */
package persistence

import (
	"database/sql"
)

// Schema 组合后的完整 Schema（供兼容使用）
// 实际执行时会分别执行 tables 和 indexes
const Schema = SchemaTables + SchemaIndexes

// InitSchema 初始化数据库 Schema
func InitSchema(db *sql.DB) error {
	// 执行表结构创建
	if _, err := db.Exec(SchemaTables); err != nil {
		return err
	}

	// 执行迁移（先建列，再建索引，避免旧库缺列时初始化失败）
	if err := migrateTasksNewColumns(db); err != nil {
		return err
	}
	if err := migrateAgentMCPBindingColumn(db); err != nil {
		return err
	}
	if err := migrateLLMProviderTypeColumn(db); err != nil {
		return err
	}
	if err := migrateDropResultColumn(db); err != nil {
		return err
	}
	if err := migrateAgentTypeColumn(db); err != nil {
		return err
	}
	if err := migrateTasksTimeoutToSeconds(db); err != nil {
		return err
	}
	if err := migrateRequirementsNewColumns(db); err != nil {
		return err
	}
	if err := migrateAgentShadowFrom(db); err != nil {
		return err
	}
	if err := migrateProjectsNewColumns(db); err != nil {
		return err
	}
	if err := migrateRequirementsClaudeRuntime(db); err != nil {
		return err
	}
	if err := migrateRequirementType(db); err != nil {
		return err
	}
	if err := migrateConversationRecordsTimestampToMillis(db); err != nil {
		return err
	}
	if err := migrateAgentLLMProviderID(db); err != nil {
		return err
	}
	if err := migrateUserTokenValue(db); err != nil {
		return err
	}

	// 所有列都补齐后再建索引，避免旧库缺列时初始化失败
	if _, err := db.Exec(SchemaIndexes); err != nil {
		return err
	}
	return nil
}

// migrateStateMachineTables 迁移状态机相关表（预留，未来可通过 migrations 调用）
func migrateStateMachineTables(db *sql.DB) error {
	// 表已通过 CREATE TABLE IF NOT EXISTS 创建
	// 此处可添加未来需要的迁移逻辑
	return nil
}

// tableHasColumn 检查表是否包含某列
func tableHasColumn(db *sql.DB, tableName, columnName string) (bool, error) {
	rows, err := db.Query("PRAGMA table_info(" + tableName + ")")
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid       int
			name      string
			columnTyp string
			notNull   int
			defaultV  sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &columnTyp, &notNull, &defaultV, &pk); err != nil {
			return false, err
		}
		if name == columnName {
			return true, nil
		}
	}
	return false, rows.Err()
}
