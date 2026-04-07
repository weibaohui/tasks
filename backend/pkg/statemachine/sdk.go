// Package statemachine 提供状态机的 SDK，支持同进程内直接调用
// 无需启动 HTTP 服务，通过一行 New() 即可使用
//
// 使用示例:
//
//	import "github.com/weibh/taskmanager/pkg/statemachine"
//
//	func main() {
//	    ctx := context.Background()
//	    sm := statemachine.New(nil) // 使用默认配置
//
//	    // 创建状态机
//	    machine, _ := sm.Create(ctx, "DevOps流程", "描述", yamlConfig)
//
//	    // 初始化需求
//	    rs, _ := sm.Initialize(ctx, "req-001", machine.ID)
//
//	    // 触发转换
//	    newState, _ := sm.Transition(ctx, "req-001", "approve", "reviewer", "通过")
//
//	    // 获取状态
//	    state, _ := sm.GetState(ctx, "req-001")
//
//	    // 获取历史
//	    history, _ := sm.GetHistory(ctx, "req-001")
//	}
package statemachine

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"

	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain/state_machine"
	"github.com/weibh/taskmanager/infrastructure/config"
	infra_sm "github.com/weibh/taskmanager/infrastructure/state_machine"
	"github.com/weibh/taskmanager/infrastructure/persistence"
	"go.uber.org/zap"
)

// SDK 状态机 SDK
type SDK struct {
	svc *application.StateMachineService
	db  *sql.DB
}

// Option SDK 配置选项
type Option func(*SDK)

// WithDB 使用自定义数据库连接
func WithDB(db *sql.DB) Option {
	return func(s *SDK) {
		s.db = db
	}
}

// WithLogger 使用自定义 logger
func WithLogger(logger *zap.Logger) Option {
	return func(s *SDK) {
		// 日志通过 StateMachineService 注入，当前忽略
	}
}

// New 创建 SDK 实例
// ctx 可以为 nil，使用默认配置
// 支持通过 Option 自定义配置
func New(ctx context.Context, opts ...Option) *SDK {
	s := &SDK{}

	// 应用配置
	for _, opt := range opts {
		opt(s)
	}

	// 如果没有提供 db，使用默认数据库
	if s.db == nil {
		dbPath := config.GetDatabasePath()
		dbDir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			panic("failed to create db dir: " + err.Error())
		}
		db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
		if err != nil {
			panic("failed to open db: " + err.Error())
		}
		s.db = db
	}

	// 创建内部服务
	repo := persistence.NewSQLiteStateMachineRepository(s.db)
	executor := infra_sm.NewTransitionExecutor(zap.NewNop())
	s.svc = application.NewStateMachineService(repo, nil, executor, zap.NewNop())

	return s
}

// Create 创建状态机
// name: 状态机名称
// description: 描述
// yamlConfig: YAML 格式的配置
func (s *SDK) Create(ctx context.Context, name, description, yamlConfig string) (*state_machine.StateMachine, error) {
	return s.svc.CreateStateMachine(ctx, name, description, yamlConfig)
}

// Get 获取状态机
func (s *SDK) Get(ctx context.Context, id string) (*state_machine.StateMachine, error) {
	return s.svc.GetStateMachine(ctx, id)
}

// List 列出所有状态机
func (s *SDK) List(ctx context.Context) ([]*state_machine.StateMachine, error) {
	return s.svc.ListStateMachines(ctx)
}

// Delete 删除状态机
func (s *SDK) Delete(ctx context.Context, id string) error {
	return s.svc.DeleteStateMachine(ctx, id)
}

// Initialize 初始化需求状态
// requirementID: 需求 ID（可以是任意字符串）
// stateMachineID: 状态机 ID
func (s *SDK) Initialize(ctx context.Context, requirementID, stateMachineID string) (*state_machine.RequirementState, error) {
	return s.svc.InitializeRequirementState(ctx, requirementID, stateMachineID)
}

// Transition 触发状态转换
// requirementID: 需求 ID
// trigger: 触发器名称
// triggeredBy: 触发者
// remark: 备注
// 可以通过 context 传递 metadata：statemachine.WithMetadata(ctx, metadata)
func (s *SDK) Transition(ctx context.Context, requirementID, trigger, triggeredBy, remark string) (*state_machine.RequirementState, error) {
	return s.svc.TriggerTransition(ctx, requirementID, trigger, triggeredBy, remark)
}

// GetState 获取需求当前状态
func (s *SDK) GetState(ctx context.Context, requirementID string) (*state_machine.RequirementState, error) {
	return s.svc.GetRequirementState(ctx, requirementID)
}

// GetHistory 获取需求转换历史
func (s *SDK) GetHistory(ctx context.Context, requirementID string) ([]*state_machine.TransitionLog, error) {
	return s.svc.GetTransitionHistory(ctx, requirementID)
}

// Close 关闭 SDK，释放资源
func (s *SDK) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// WithMetadata 将 metadata 存入 context，用于 Hook 模板变量替换
// 用法: sm.Transition(statemachine.WithMetadata(ctx, metadata), ...)
func WithMetadata(ctx context.Context, metadata map[string]interface{}) context.Context {
	return infra_sm.WithMetadata(ctx, metadata)
}
