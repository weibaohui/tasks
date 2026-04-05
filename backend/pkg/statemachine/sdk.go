// Package statemachine 提供状态机的 SDK，支持同进程内直接调用
// 无需启动 HTTP 服务，通过一行 New() 即可使用
//
// 使用示例:
//
//	import "github.com/weibh/taskmanager/pkg/statemachine"
//
//	func main() {
//	    sm := statemachine.New() // 自动管理依赖
//
//	    machine, _ := sm.Create(ctx, "DevOps流程", "描述", yamlConfig)
//	    rs, _ := sm.Initialize(ctx, "req-001", machine.ID)
//	    rs, _ = sm.Transition(ctx, "req-001", "approve", "reviewer", "通过")
//
//	    sm.Close()
//	}
//
// 也支持依赖注入:
//
//	import "go.uber.org/zap"
//
//	logger := zap.NewExample()
//	db, _ := sql.Open("sqlite3", "file::memory:?cache=shared")
//	sm := statemachine.New(
//	    statemachine.WithDB(db),
//	    statemachine.WithLogger(logger),
//	)
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
type Option func(*sdkOptions)

type sdkOptions struct {
	db     *sql.DB
	logger *zap.Logger
	svc    *application.StateMachineService
}

// WithDB 使用自定义数据库连接
func WithDB(db *sql.DB) Option {
	return func(o *sdkOptions) {
		o.db = db
	}
}

// WithLogger 使用自定义 logger
func WithLogger(logger *zap.Logger) Option {
	return func(o *sdkOptions) {
		o.logger = logger
	}
}

// WithService 使用已有的 StateMachineService（用于测试或已创建的服务）
func WithService(svc *application.StateMachineService) Option {
	return func(o *sdkOptions) {
		o.svc = svc
	}
}

// New 创建 SDK 实例
// 选项：
//   - WithDB: 自定义数据库连接
//   - WithLogger: 自定义日志器
//   - WithService: 直接注入已创建的服务
func New(opts ...Option) *SDK {
	o := &sdkOptions{
		logger: zap.NewNop(),
	}

	// 应用选项
	for _, opt := range opts {
		opt(o)
	}

	s := &SDK{}

	// 如果注入了服务，直接使用
	if o.svc != nil {
		s.svc = o.svc
		return s
	}

	// 否则自动创建依赖
	if o.db == nil {
		dbPath := config.GetDatabasePath()
		dbDir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			panic("failed to create db dir: " + err.Error())
		}
		db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
		if err != nil {
			panic("failed to open db: " + err.Error())
		}
		o.db = db
	}

	repo := persistence.NewSQLiteStateMachineRepository(o.db)
	executor := infra_sm.NewTransitionExecutor(o.logger)
	s.svc = application.NewStateMachineService(repo, executor, o.logger)
	s.db = o.db

	return s
}

// Create 创建状态机
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
func (s *SDK) Initialize(ctx context.Context, requirementID, stateMachineID string) (*state_machine.RequirementState, error) {
	return s.svc.InitializeRequirementState(ctx, requirementID, stateMachineID)
}

// Transition 触发状态转换
// 可以通过 statemachine.WithMetadata(ctx, metadata) 传递元数据
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
// 只有 SDK 自己创建的 db 才会被关闭，通过 WithDB 注入的不会
func (s *SDK) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// WithMetadata 将 metadata 存入 context，用于 Hook 模板变量替换
func WithMetadata(ctx context.Context, metadata map[string]interface{}) context.Context {
	return infra_sm.WithMetadata(ctx, metadata)
}
