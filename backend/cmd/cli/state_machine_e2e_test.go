package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/weibh/taskmanager/application"
	_persistence "github.com/weibh/taskmanager/infrastructure/persistence"
	infra_sm "github.com/weibh/taskmanager/infrastructure/state_machine"
	"go.uber.org/zap"
)

// TestStateMachineE2E 状态机端到端测试
func TestStateMachineE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过 E2E 测试 (使用 -short 标志)")
	}

	// 创建临时数据库
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("无法创建数据库: %v", err)
	}
	defer db.Close()

	if err := _persistence.InitSchema(db); err != nil {
		t.Fatalf("初始化 Schema 失败: %v", err)
	}

	logger, _ := zap.NewDevelopment()

	// 创建仓储
	repo := _persistence.NewSQLiteStateMachineRepository(db)

	// 创建 Hook 执行器
	executor := infra_sm.NewTransitionExecutor(logger)

	// 创建服务
	svc := application.NewStateMachineService(repo, executor, logger)
	ctx := context.Background()

	// 测试完整的流程
	t.Run("完整流程测试", func(t *testing.T) {
		// 1. 创建状态机
		sm, err := svc.CreateStateMachine(ctx, "project-1", "test_flow", "测试流程", testFlowYAML)
		if err != nil {
			t.Fatalf("创建状态机失败: %v", err)
		}
		t.Logf("创建状态机成功: %s", sm.ID)

		// 2. 绑定类型
		err = svc.BindType(ctx, sm.ID, "normal")
		if err != nil {
			t.Fatalf("绑定类型失败: %v", err)
		}
		t.Log("绑定类型成功")

		// 3. 创建需求状态
		rs, err := svc.InitializeRequirementState(ctx, "req-001", "project-1", "normal")
		if err != nil {
			t.Fatalf("初始化需求状态失败: %v", err)
		}
		if rs.CurrentState != "created" {
			t.Errorf("期望初始状态为 created, 实际为 %s", rs.CurrentState)
		}
		t.Logf("需求状态初始化成功: %s", rs.CurrentState)

		// 4. 触发转换: created -> in_progress
		rs, err = svc.TriggerTransition(ctx, "req-001", "start", "user", "开始处理")
		if err != nil {
			t.Fatalf("触发转换失败: %v", err)
		}
		if rs.CurrentState != "in_progress" {
			t.Errorf("期望状态为 in_progress, 实际为 %s", rs.CurrentState)
		}
		t.Logf("转换成功: %s", rs.CurrentState)

		// 5. 获取转换历史
		logs, err := svc.GetTransitionHistory(ctx, "req-001")
		if err != nil {
			t.Fatalf("获取转换历史失败: %v", err)
		}
		if len(logs) != 2 { // init + start
			t.Errorf("期望2条日志, 实际为 %d", len(logs))
		}
		t.Logf("转换历史: %d 条", len(logs))

		// 6. 触发转换: in_progress -> completed
		rs, err = svc.TriggerTransition(ctx, "req-001", "complete", "user", "完成")
		if err != nil {
			t.Fatalf("触发转换失败: %v", err)
		}
		if rs.CurrentState != "completed" {
			t.Errorf("期望状态为 completed, 实际为 %s", rs.CurrentState)
		}
		t.Logf("转换成功: %s", rs.CurrentState)
	})

	t.Run("无效转换测试", func(t *testing.T) {
		// 尝试在终态上触发转换
		_, err := svc.TriggerTransition(ctx, "req-001", "start", "user", "")
		if err == nil {
			t.Error("期望转换失败：终态不能转换")
		}
		t.Logf("终态转换被正确拒绝: %v", err)
	})

	t.Run("无效触发器测试", func(t *testing.T) {
		// 重新初始化一个需求
		rs, _ := svc.InitializeRequirementState(ctx, "req-002", "project-1", "normal")

		// 尝试无效的触发器
		_, err := svc.TriggerTransition(ctx, "req-002", "invalid_trigger", "user", "")
		if err == nil {
			t.Error("期望转换失败：无效触发器")
		}
		t.Logf("无效触发器被正确拒绝: %v", err)

		// 清理
		repo.DeleteStateMachine(ctx, rs.StateMachineID)
	})
}

// TestTransitionHookE2E 转换 Hook 端到端测试
func TestTransitionHookE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过 E2E 测试 (使用 -short 标志)")
	}

	// 创建临时数据库
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("无法创建数据库: %v", err)
	}
	defer db.Close()

	if err := _persistence.InitSchema(db); err != nil {
		t.Fatalf("初始化 Schema 失败: %v", err)
	}

	logger, _ := zap.NewDevelopment()

	// 创建模拟 HTTP 服务器来接收 hook 调用
	var hookCalls []map[string]interface{}
	var hookMu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", 400)
			return
		}
		hookMu.Lock()
		hookCalls = append(hookCalls, req)
		hookMu.Unlock()
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	// 创建带 hook 的状态机 YAML
	yamlWithHook := `
name: hook_test_flow
description: Hook 测试流程
initial_state: created

states:
  - id: created
    name: 已创建
    is_final: false
  - id: in_progress
    name: 进行中
    is_final: false
  - id: completed
    name: 已完成
    is_final: true

transitions:
  - from: created
    to: in_progress
    trigger: start
    description: 开始
    hooks:
      - name: 通知开始
        type: webhook
        config:
          url: "` + server.URL + `"
          method: POST
        retry: 1
        timeout: 30

  - from: in_progress
    to: completed
    trigger: complete
    description: 完成
    hooks:
      - name: 通知完成
        type: webhook
        config:
          url: "` + server.URL + `"
          method: POST
        retry: 1
        timeout: 30
`

	// 创建仓储
	repo := _persistence.NewSQLiteStateMachineRepository(db)

	// 创建 Hook 执行器
	executor := infra_sm.NewTransitionExecutor(logger)

	// 创建服务
	svc := application.NewStateMachineService(repo, executor, logger)
	ctx := context.Background()

	// 创建状态机
	sm, err := svc.CreateStateMachine(ctx, "project-1", "hook_test", "Hook测试", yamlWithHook)
	if err != nil {
		t.Fatalf("创建状态机失败: %v", err)
	}

	// 绑定类型
	svc.BindType(ctx, sm.ID, "normal")

	// 初始化需求
	svc.InitializeRequirementState(ctx, "req-hook-001", "project-1", "normal")

	// 触发第一次转换
	svc.TriggerTransition(ctx, "req-hook-001", "start", "user", "开始")

	// 等待异步 hook 执行
	time.Sleep(500 * time.Millisecond)

	// 验证 hook 被调用
	hookMu.Lock()
	if len(hookCalls) < 1 {
		t.Errorf("期望至少1次 hook 调用, 实际为 %d", len(hookCalls))
	}
	if len(hookCalls) > 0 {
		t.Logf("Hook 调用次数: %d", len(hookCalls))
		for i, call := range hookCalls {
			t.Logf("Hook #%d: requirement_id=%v, hook_name=%v", i+1, call["requirement_id"], call["hook_name"])
		}
	}
	hookMu.Unlock()

	// 触发第二次转换
	svc.TriggerTransition(ctx, "req-hook-001", "complete", "user", "完成")

	// 等待异步 hook 执行
	time.Sleep(500 * time.Millisecond)

	// 验证第二个 hook 被调用
	hookMu.Lock()
	if len(hookCalls) < 2 {
		t.Errorf("期望至少2次 hook 调用, 实际为 %d", len(hookCalls))
	}
	hookMu.Unlock()

	t.Logf("Hook E2E 测试完成: %d 次调用", len(hookCalls))
}

// TestHeartbeatStateMachineE2E 心跳状态机端到端测试
func TestHeartbeatStateMachineE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过 E2E 测试 (使用 -short 标志)")
	}

	// 创建临时数据库
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("无法创建数据库: %v", err)
	}
	defer db.Close()

	if err := _persistence.InitSchema(db); err != nil {
		t.Fatalf("初始化 Schema 失败: %v", err)
	}

	logger, _ := zap.NewDevelopment()

	// 创建仓储
	repo := _persistence.NewSQLiteStateMachineRepository(db)

	// 创建 Hook 执行器
	executor := infra_sm.NewTransitionExecutor(logger)

	// 创建服务
	svc := application.NewStateMachineService(repo, executor, logger)
	ctx := context.Background()

	// 心跳状态机 YAML
	heartbeatYAML := `
name: heartbeat_flow
description: 心跳任务流程
initial_state: active

states:
  - id: active
    name: 活跃
    is_final: false
  - id: stopped
    name: 已停止
    is_final: false

transitions:
  - from: active
    to: stopped
    trigger: stop
    description: 停止心跳

  - from: stopped
    to: active
    trigger: restart
    description: 重启心跳
`

	// 创建状态机
	sm, err := svc.CreateStateMachine(ctx, "project-1", "heartbeat", "心跳流程", heartbeatYAML)
	if err != nil {
		t.Fatalf("创建状态机失败: %v", err)
	}

	// 绑定 heartbeat 类型
	err = svc.BindType(ctx, sm.ID, "heartbeat")
	if err != nil {
		t.Fatalf("绑定类型失败: %v", err)
	}

	// 初始化心跳需求
	rs, err := svc.InitializeRequirementState(ctx, "heartbeat-001", "project-1", "heartbeat")
	if err != nil {
		t.Fatalf("初始化需求状态失败: %v", err)
	}

	if rs.CurrentState != "active" {
		t.Errorf("期望初始状态为 active, 实际为 %s", rs.CurrentState)
	}

	// 停止心跳
	rs, err = svc.TriggerTransition(ctx, "heartbeat-001", "stop", "system", "停止")
	if err != nil {
		t.Fatalf("停止心跳失败: %v", err)
	}

	if rs.CurrentState != "stopped" {
		t.Errorf("期望状态为 stopped, 实际为 %s", rs.CurrentState)
	}

	// 重启心跳
	rs, err = svc.TriggerTransition(ctx, "heartbeat-001", "restart", "system", "重启")
	if err != nil {
		t.Fatalf("重启心跳失败: %v", err)
	}

	if rs.CurrentState != "active" {
		t.Errorf("期望状态为 active, 实际为 %s", rs.CurrentState)
	}

	t.Log("心跳状态机 E2E 测试通过")
}

const testFlowYAML = `
name: test_flow
description: 测试流程
initial_state: created

states:
  - id: created
    name: 已创建
    is_final: false
  - id: in_progress
    name: 进行中
    is_final: false
  - id: completed
    name: 已完成
    is_final: true

transitions:
  - from: created
    to: in_progress
    trigger: start
    description: 开始处理

  - from: in_progress
    to: completed
    trigger: complete
    description: 完成
`
