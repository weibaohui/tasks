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

// MigrateDefaultAgentCodeColumn 兼容旧数据库：在 projects 表中添加 default_agent_code 列
func MigrateDefaultAgentCodeColumn(db *sql.DB) error {
	columns, err := getTableColumns(db, "projects")
	if err != nil {
		return fmt.Errorf("获取 projects 表列信息失败: %w", err)
	}

	if _, exists := columns["default_agent_code"]; !exists {
		if _, err := db.Exec("ALTER TABLE projects ADD COLUMN default_agent_code TEXT NOT NULL DEFAULT ''"); err != nil {
			return fmt.Errorf("添加 default_agent_code 列失败: %w", err)
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

	// 4. 迁移完成后删除旧列，避免重复迁移
	dropColumns := []string{
		"heartbeat_enabled",
		"heartbeat_interval_minutes",
		"heartbeat_md_content",
		"agent_code",
	}
	for _, col := range dropColumns {
		if _, exists := columns[col]; exists {
			if _, err := db.Exec(fmt.Sprintf("ALTER TABLE projects DROP COLUMN %s", col)); err != nil {
				return fmt.Errorf("删除旧列 %s 失败: %w", col, err)
			}
		}
	}

	return nil
}

// MigrateHeartbeatScenarioCodeColumn 兼容旧数据库：在 projects 表中添加 heartbeat_scenario_code 列
func MigrateHeartbeatScenarioCodeColumn(db *sql.DB) error {
	columns, err := getTableColumns(db, "projects")
	if err != nil {
		return fmt.Errorf("获取 projects 表列信息失败: %w", err)
	}

	if _, exists := columns["heartbeat_scenario_code"]; !exists {
		if _, err := db.Exec("ALTER TABLE projects ADD COLUMN heartbeat_scenario_code TEXT NOT NULL DEFAULT ''"); err != nil {
			return fmt.Errorf("添加 heartbeat_scenario_code 列失败: %w", err)
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

// MigrateGitHubWebhookConfigColumns 兼容旧数据库：将 github_webhook_configs 表的 forwarder_pid 列改为 webhook_url 列
func MigrateGitHubWebhookConfigColumns(db *sql.DB) error {
	columns, err := getTableColumns(db, "github_webhook_configs")
	if err != nil {
		return fmt.Errorf("获取 github_webhook_configs 表列信息失败: %w", err)
	}

	// 如果存在旧的 forwarder_pid 列，需要迁移
	if _, exists := columns["forwarder_pid"]; exists {
		// 添加 webhook_url 列（如果不存在）
		if _, exists := columns["webhook_url"]; !exists {
			if _, err := db.Exec("ALTER TABLE github_webhook_configs ADD COLUMN webhook_url TEXT"); err != nil {
				return fmt.Errorf("添加 webhook_url 列失败: %w", err)
			}
		}
		// 注意：SQLite 不支持直接删除列，需要重建表才能移除 forwarder_pid
		// 为了保持向后兼容，保留 forwarder_pid 列，只是应用程序不再使用它
	}

	return nil
}

// MigrateWebhookEventTriggeredHeartbeatsTable 创建 webhook_event_triggered_heartbeats 表（如果不存在）
func MigrateWebhookEventTriggeredHeartbeatsTable(db *sql.DB) error {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS webhook_event_triggered_heartbeats (
		id TEXT PRIMARY KEY,
		webhook_event_log_id TEXT NOT NULL,
		heartbeat_id TEXT NOT NULL,
		requirement_id TEXT,
		triggered_at INTEGER NOT NULL,
		FOREIGN KEY (webhook_event_log_id) REFERENCES webhook_event_logs(id) ON DELETE CASCADE,
		FOREIGN KEY (heartbeat_id) REFERENCES heartbeats(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_webhook_event_triggered_heartbeats_event_id ON webhook_event_triggered_heartbeats(webhook_event_log_id);
	`
	if _, err := db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("创建 webhook_event_triggered_heartbeats 表失败: %w", err)
	}
	return nil
}

// MigrateWebhookEventLogsAddColumns 为 webhook_event_logs 表添加缺失的列
func MigrateWebhookEventLogsAddColumns(db *sql.DB) error {
	columns, err := getTableColumns(db, "webhook_event_logs")
	if err != nil {
		return fmt.Errorf("获取 webhook_event_logs 表列信息失败: %w", err)
	}

	alterStatements := []struct {
		column string
		sql    string
	}{
		{"method", "ALTER TABLE webhook_event_logs ADD COLUMN method TEXT NOT NULL DEFAULT ''"},
		{"headers", "ALTER TABLE webhook_event_logs ADD COLUMN headers TEXT NOT NULL DEFAULT ''"},
		{"requirement_id", "ALTER TABLE webhook_event_logs ADD COLUMN requirement_id TEXT"},
	}

	for _, stmt := range alterStatements {
		if _, exists := columns[stmt.column]; !exists {
			if _, err := db.Exec(stmt.sql); err != nil {
				return fmt.Errorf("添加 %s 列失败: %w", stmt.column, err)
			}
		}
	}

	return nil
}

// SeedHeartbeatTemplates 预置默认心跳模板（如果表为空）
func SeedHeartbeatTemplates(db *sql.DB) error {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM heartbeat_templates").Scan(&count); err != nil {
		return fmt.Errorf("查询心跳模板数量失败: %w", err)
	}
	if count > 0 {
		return nil
	}

	templates := []struct {
		id, name, mdContent, requirementType string
	}{
		{
			id:              "ht_default_normal",
			name:            "派发需求",
			requirementType: "normal",
			mdContent: `你是一个心跳调度员，你的任务是：需求派发（normal 类型）。

切记，你是调度员，不是 CodingAgent。
- 你严禁修改任何源代码
- 你严禁执行 git commit、git push、gh pr create
- 你需要处理的内容，必须通过 taskmanager requirement create 或 gh 命令完成，形成明确的需求或流程动作
- 创建需求时必须指定 --type 参数：
  - 普通需求：--type normal

# 任务：派发需求
## 1.1 查看需求列表
使用 taskmanager requirement list 命令查看当前未处理的需求列表，找到状态为 status=todo，requirement_type=normal 的需求，并按创建时间排序。
## 1.2 派发需求
派发第一个待处理的需求。命令示例：taskmanager requirement dispatch <requirement_id>
## 1.3 派发注意事项
- 只派发 todo 状态的需求
- 已完成、进行中、失败的需求不要派发
- 每次心跳最多派发一个需求
- 优先派发最早创建的需求`,
		},
		{
			id:              "ht_default_pr_review",
			name:            "处理 PR",
			requirementType: "pr_review",
			mdContent: `你是一个心跳调度员，你的任务是：处理PR（pr_review 类型）。

切记，你是调度员，不是 CodingAgent。
- 你严禁修改任何源代码
- 你严禁执行 git commit、git push、gh pr create
- 你需要处理的内容，必须通过 taskmanager requirement create 或 gh 命令完成，形成明确的需求或流程动作
- 创建需求时必须指定 --type 参数：
  - PR修复需求：--type pr_review

# 任务：处理PR
## 1. 获取待合并的PR列表
gh pr list --state open --mergeable non-conflicting --json number,title,author,body,url

## 2. 分析每个PR
对于每个待合并的PR：
1. 对于所有评论已解决、CI通过、代码审查通过的PR，可以评论 /lgtm。使用 gh pr comment <PR_NUMBER> --body "/lgtm"。
2. 对于已经有 /lgtm 的评论，可以直接合并到 main 分支，并删除源分支：gh pr merge <PR_NUMBER> --squash --delete-branch
3. 你判断 reviewer 提出的评论建议是否需要修复，如需修复，请创建需求让另一AI执行修复；如不需要修复，直接评论 /lgtm。注意你不要自己修复。
4. 创建代码修复需求
使用 taskmanager requirement create --project-id <PROJECT_ID> --type pr_review --title "[PR修复] <修复标题>" --description "# 任务目标：修复问题，要使用有问题的分支，严禁创建新分支。## 背景来源：PR #<PR号> 评论：<reviewer评论内容及摘要> ## 修复分支 请进入<branch_name>进行修复，修复完成后提交并推送该分支。" --acceptance "具体验收标准"

## 3. PR处理重要原则
1. 需求必须独立可完成
2. 描述要让AI无需再看PR就能工作
3. 代码修复要写清在哪个分支、哪个文件、具体修复内容
4. 仅在必要时创建需求
5. 没问题的PR写 /lgtm`,
		},
		{
			id:              "ht_default_optimization",
			name:            "提出优化点",
			requirementType: "optimization",
			mdContent: `你是一个心跳调度员，你的任务是：提出优化点（optimization 类型）。

切记，你是调度员，不是 CodingAgent。
- 你严禁修改任何源代码
- 你严禁执行 git commit、git push、gh pr create
- 你需要处理的内容，必须通过 taskmanager requirement create 或 gh 命令完成，形成明确的需求或流程动作
- 创建需求时必须指定 --type 参数：
  - 优化需求：--type optimization

# 任务：提出优化点
当没有工作可干的时候，你按下面的工作方向，任选其一。
## 3.1 工作方向（每次心跳任选其一）
1. 按Go最佳实践，检查各个模块，对于需要优化的文件，使用 taskmanager requirement create --type optimization 生成针对某个方面的具体的优化需求。
2. 检查测试用例情况，如果你觉得需要某个测试不够好，使用 taskmanager requirement create --type optimization 生成测试用例编写需求。
3. 搜索代码，找出可以优化的功能点，使用 taskmanager requirement create --type optimization 生成具体的功能需求。`,
		},
	}

	now := time.Now().Unix()
	for _, t := range templates {
		_, err := db.Exec(`
			INSERT INTO heartbeat_templates (id, name, md_content, requirement_type, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, t.id, t.name, t.mdContent, t.requirementType, now, now)
		if err != nil {
			return fmt.Errorf("插入默认模板 %s 失败: %w", t.id, err)
		}
	}
	return nil
}
