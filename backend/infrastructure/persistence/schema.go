/**
 * 数据库 Schema 定义
 */
package persistence

import "database/sql"

// Schema SQL Schema 定义
const Schema = `
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    trace_id TEXT NOT NULL,
    span_id TEXT NOT NULL,
    parent_id TEXT,
    name TEXT NOT NULL,
    description TEXT,
    type TEXT NOT NULL,
    metadata TEXT,
    timeout INTEGER NOT NULL,
    max_retries INTEGER NOT NULL,
    priority INTEGER NOT NULL,
    status INTEGER NOT NULL,
    progress TEXT,
    result TEXT,
    error_msg TEXT,
    created_at INTEGER NOT NULL,
    started_at INTEGER,
    finished_at INTEGER
);

CREATE INDEX IF NOT EXISTS idx_tasks_trace_id ON tasks(trace_id);
CREATE INDEX IF NOT EXISTS idx_tasks_parent_id ON tasks(parent_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);

CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    user_code TEXT NOT NULL UNIQUE,
    username TEXT NOT NULL UNIQUE,
    email TEXT,
    display_name TEXT,
    password_hash TEXT NOT NULL,
    is_active INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_user_code ON users(user_code);
`

// InitSchema 初始化数据库 Schema
func InitSchema(db *sql.DB) error {
	_, err := db.Exec(Schema)
	return err
}
