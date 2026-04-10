package statemachine

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

const testYAML = `
name: 测试流程
description: SDK 测试用流程

initial_state: created

states:
  - id: todo
    name: 待办
    is_final: false
  - id: created
    name: 已创建
    is_final: false
  - id: in_progress
    name: 进行中
    is_final: false
  - id: done
    name: 已完成
    is_final: false
  - id: completed
    name: 已完成
    is_final: true

transitions:
  - from: todo
    to: created
    trigger: create
    description: 创建
  - from: created
    to: in_progress
    trigger: start
    description: 开始
  - from: in_progress
    to: done
    trigger: complete
    description: 完成
  - from: done
    to: completed
    trigger: finish
    description: 结束
`

func TestSDK(t *testing.T) {
	// 创建临时数据库
	tmpFile, err := os.CreateTemp("", "statemachine-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	db, err := sql.Open("sqlite3", tmpFile.Name()+"?_journal_mode=WAL")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// 初始化表结构
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS state_machines (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			config TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS requirement_states (
			id TEXT PRIMARY KEY,
			requirement_id TEXT NOT NULL UNIQUE,
			state_machine_id TEXT NOT NULL,
			current_state TEXT NOT NULL,
			current_state_name TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS transition_logs (
			id TEXT PRIMARY KEY,
			requirement_id TEXT NOT NULL,
			from_state TEXT NOT NULL,
			to_state TEXT NOT NULL,
			trigger TEXT NOT NULL,
			triggered_by TEXT NOT NULL,
			remark TEXT,
			result TEXT NOT NULL,
			error_message TEXT,
			created_at INTEGER NOT NULL
		);
	`); err != nil {
		t.Fatal(err)
	}

	// 创建 SDK
	sm := New(context.Background(), WithDB(db))
	defer sm.Close()

	ctx := context.Background()

	// 创建状态机
	t.Log("创建状态机...")
	machine, err := sm.Create(ctx, "测试流程", "测试", testYAML)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	t.Logf("状态机创建成功: ID=%s", machine.ID)

	// 获取状态机
	t.Log("获取状态机...")
	got, err := sm.Get(ctx, machine.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != machine.ID {
		t.Errorf("Get returned wrong ID: %s != %s", got.ID, machine.ID)
	}

	// 列出状态机
	t.Log("列出状态机...")
	list, err := sm.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("List returned %d items, want 1", len(list))
	}

	// 初始化需求
	t.Log("初始化需求...")
	reqID := "test-req-001"
	rs, err := sm.Initialize(ctx, reqID, machine.ID)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if rs.CurrentState != "created" {
		t.Errorf("Initial state: %s != created", rs.CurrentState)
	}

	// 触发转换 start
	t.Log("触发转换 start...")
	rs, err = sm.Transition(ctx, reqID, "start", "tester", "开始测试")
	if err != nil {
		t.Fatalf("Transition(start) failed: %v", err)
	}
	if rs.CurrentState != "in_progress" {
		t.Errorf("State after start: %s != in_progress", rs.CurrentState)
	}

	// 触发转换 complete
	t.Log("触发转换 complete...")
	rs, err = sm.Transition(ctx, reqID, "complete", "tester", "完成测试")
	if err != nil {
		t.Fatalf("Transition(complete) failed: %v", err)
	}
	if rs.CurrentState != "done" {
		t.Errorf("State after complete: %s != done", rs.CurrentState)
	}

	// 获取状态
	t.Log("获取状态...")
	state, err := sm.GetState(ctx, reqID)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if state.CurrentState != "done" {
		t.Errorf("GetState state: %s != done", state.CurrentState)
	}

	// 获取历史
	t.Log("获取历史...")
	history, err := sm.GetHistory(ctx, reqID)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}
	// init + start + complete = 3
	if len(history) != 3 {
		t.Errorf("History count: %d != 3", len(history))
	}

	// 删除状态机
	t.Log("删除状态机...")
	err = sm.Delete(ctx, machine.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	t.Log("测试完成!")
}
