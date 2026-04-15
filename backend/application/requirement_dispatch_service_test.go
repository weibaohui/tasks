package application

import (
	"context"
	"testing"

	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/domain/statemachine"
	channelBus "github.com/weibh/taskmanager/pkg/bus"
)

func TestResolveReplicaAgentCwd(t *testing.T) {
	workspacePath := "/tmp/ai-devops/proj-001/req-001"

	requirementWithTemp, err := domain.NewRequirement(
		domain.NewRequirementID("req-001"),
		domain.NewProjectID("proj-001"),
		"需求标题",
		"需求描述",
		"验收标准",
		" /tmp/custom-workspace ",
	)
	if err != nil {
		t.Fatalf("创建需求失败: %v", err)
	}

	if got := resolveReplicaAgentCwd(requirementWithTemp, workspacePath); got != workspacePath {
		t.Fatalf("期望始终使用完整派发工作目录，实际为: %s", got)
	}

	requirementWithoutTemp, err := domain.NewRequirement(
		domain.NewRequirementID("req-002"),
		domain.NewProjectID("proj-001"),
		"需求标题",
		"需求描述",
		"验收标准",
		"",
	)
	if err != nil {
		t.Fatalf("创建需求失败: %v", err)
	}

	if got := resolveReplicaAgentCwd(requirementWithoutTemp, workspacePath); got != workspacePath {
		t.Fatalf("期望回退到派发工作目录，实际为: %s", got)
	}

	if got := resolveReplicaAgentCwd(nil, workspacePath); got != workspacePath {
		t.Fatalf("期望 nil 需求回退到派发工作目录，实际为: %s", got)
	}
}

type mockRequirementRepoForDispatch struct {
	requirements map[string]*domain.Requirement
	countResult  int
	countErr     error
}

func (m *mockRequirementRepoForDispatch) Save(ctx context.Context, requirement *domain.Requirement) error {
	m.requirements[requirement.ID().String()] = requirement
	return nil
}

func (m *mockRequirementRepoForDispatch) FindByID(ctx context.Context, id domain.RequirementID) (*domain.Requirement, error) {
	r, ok := m.requirements[id.String()]
	if !ok {
		return nil, nil
	}
	return r, nil
}

func (m *mockRequirementRepoForDispatch) FindByProjectID(ctx context.Context, projectID domain.ProjectID) ([]*domain.Requirement, error) {
	return nil, nil
}

func (m *mockRequirementRepoForDispatch) FindByTraceID(ctx context.Context, traceID string) (*domain.Requirement, error) {
	return nil, nil
}

func (m *mockRequirementRepoForDispatch) FindAll(ctx context.Context) ([]*domain.Requirement, error) {
	return nil, nil
}

func (m *mockRequirementRepoForDispatch) List(ctx context.Context, filter domain.RequirementListFilter) ([]*domain.Requirement, error) {
	return nil, nil
}

func (m *mockRequirementRepoForDispatch) Count(ctx context.Context, filter domain.RequirementListFilter) (int, error) {
	return m.countResult, m.countErr
}

func (m *mockRequirementRepoForDispatch) Delete(ctx context.Context, id domain.RequirementID) error {
	return nil
}

func (m *mockRequirementRepoForDispatch) GetStatusStats(ctx context.Context, projectID *domain.ProjectID) ([]domain.StatusStat, error) {
	return nil, nil
}

type mockAgentRepoForDispatch struct {
	agents map[string]*domain.Agent
}

func (m *mockAgentRepoForDispatch) Save(ctx context.Context, agent *domain.Agent) error {
	return nil
}

func (m *mockAgentRepoForDispatch) FindByID(ctx context.Context, id domain.AgentID) (*domain.Agent, error) {
	return nil, nil
}

func (m *mockAgentRepoForDispatch) FindByAgentCode(ctx context.Context, code domain.AgentCode) (*domain.Agent, error) {
	a, ok := m.agents[code.String()]
	if !ok {
		return nil, nil
	}
	return a, nil
}

func (m *mockAgentRepoForDispatch) FindByUserCode(ctx context.Context, userCode string) ([]*domain.Agent, error) {
	return nil, nil
}

func (m *mockAgentRepoForDispatch) FindAll(ctx context.Context) ([]*domain.Agent, error) {
	return nil, nil
}

func (m *mockAgentRepoForDispatch) Delete(ctx context.Context, id domain.AgentID) error {
	return nil
}

type mockReplicaCleanupService struct{}

func (m *mockReplicaCleanupService) CleanupReplica(ctx context.Context, replicaAgentCode, workspacePath string) error {
	return nil
}

type mockStateMachineRepoForDispatch struct{}

func (m *mockStateMachineRepoForDispatch) SaveStateMachine(ctx context.Context, sm *statemachine.StateMachine) error {
	return nil
}
func (m *mockStateMachineRepoForDispatch) GetStateMachine(ctx context.Context, id string) (*statemachine.StateMachine, error) {
	return nil, nil
}
func (m *mockStateMachineRepoForDispatch) ListStateMachines(ctx context.Context) ([]*statemachine.StateMachine, error) {
	return nil, nil
}
func (m *mockStateMachineRepoForDispatch) DeleteStateMachine(ctx context.Context, id string) error {
	return nil
}
func (m *mockStateMachineRepoForDispatch) SaveRequirementState(ctx context.Context, rs *statemachine.RequirementState) error {
	return nil
}
func (m *mockStateMachineRepoForDispatch) GetRequirementState(ctx context.Context, requirementID string) (*statemachine.RequirementState, error) {
	return nil, nil
}
func (m *mockStateMachineRepoForDispatch) UpdateRequirementState(ctx context.Context, rs *statemachine.RequirementState) error {
	return nil
}
func (m *mockStateMachineRepoForDispatch) SaveTransitionLog(ctx context.Context, log *statemachine.TransitionLog) error {
	return nil
}
func (m *mockStateMachineRepoForDispatch) ListTransitionLogs(ctx context.Context, requirementID string) ([]*statemachine.TransitionLog, error) {
	return nil, nil
}
func (m *mockStateMachineRepoForDispatch) SaveProjectStateMachine(ctx context.Context, psm *statemachine.ProjectStateMachine) error {
	return nil
}
func (m *mockStateMachineRepoForDispatch) GetProjectStateMachine(ctx context.Context, projectID string, requirementType statemachine.RequirementType) (*statemachine.ProjectStateMachine, error) {
	return nil, nil
}
func (m *mockStateMachineRepoForDispatch) ListProjectStateMachines(ctx context.Context, projectID string) ([]*statemachine.ProjectStateMachine, error) {
	return nil, nil
}
func (m *mockStateMachineRepoForDispatch) DeleteProjectStateMachine(ctx context.Context, id string) error {
	return nil
}
func (m *mockStateMachineRepoForDispatch) DeleteProjectStateMachinesByProject(ctx context.Context, projectID string) error {
	return nil
}

type mockWorkspaceConfigForDispatch struct{}

func (m *mockWorkspaceConfigForDispatch) WorkspaceRoot() string { return "/tmp/workspace" }

type mockWorkspaceManagerForDispatch struct{}

func (m *mockWorkspaceManagerForDispatch) CreateWorkspace(path string) error { return nil }
func (m *mockWorkspaceManagerForDispatch) RemoveWorkspace(path string) error { return nil }
func (m *mockWorkspaceManagerForDispatch) WorkspaceExists(path string) bool   { return false }

type mockInboundPublisherForDispatch struct {
	published bool
}

func (m *mockInboundPublisherForDispatch) PublishInbound(msg *channelBus.InboundMessage) {
	m.published = true
}

type mockSessionRepoForDispatch struct {
	sessions map[string]*domain.Session
}

func (m *mockSessionRepoForDispatch) Save(ctx context.Context, session *domain.Session) error {
	return nil
}
func (m *mockSessionRepoForDispatch) FindByID(ctx context.Context, id domain.SessionID) (*domain.Session, error) {
	return nil, nil
}
func (m *mockSessionRepoForDispatch) FindBySessionKey(ctx context.Context, sessionKey string) (*domain.Session, error) {
	s, ok := m.sessions[sessionKey]
	if !ok {
		return nil, nil
	}
	return s, nil
}
func (m *mockSessionRepoForDispatch) FindByUserCode(ctx context.Context, userCode string) ([]*domain.Session, error) {
	return nil, nil
}
func (m *mockSessionRepoForDispatch) FindByChannelCode(ctx context.Context, channelCode string) ([]*domain.Session, error) {
	return nil, nil
}
func (m *mockSessionRepoForDispatch) FindActiveByUserCode(ctx context.Context, userCode string) ([]*domain.Session, error) {
	return nil, nil
}
func (m *mockSessionRepoForDispatch) DeleteBySessionKey(ctx context.Context, sessionKey string) error {
	return nil
}
func (m *mockSessionRepoForDispatch) DeleteByChannelCode(ctx context.Context, channelCode string) error {
	return nil
}

type mockIDGeneratorForDispatch struct {
	counter int
}

func (m *mockIDGeneratorForDispatch) Generate() string {
	m.counter++
	return "dispatch-id-" + string(rune('0'+m.counter))
}

func setupDispatchService(countResult int, countErr error) (*RequirementDispatchService, *mockRequirementRepoForDispatch) {
	reqRepo := &mockRequirementRepoForDispatch{
		requirements: make(map[string]*domain.Requirement),
		countResult:  countResult,
		countErr:     countErr,
	}
	projectRepo := newSharedMockProjectRepo()
	agentRepo := &mockAgentRepoForDispatch{agents: make(map[string]*domain.Agent)}
	idGen := &mockIDGeneratorForDispatch{counter: 0}
	publisher := &mockInboundPublisherForDispatch{}

	sessionRepo := &mockSessionRepoForDispatch{
		sessions: map[string]*domain.Session{
			"test:chat123": func() *domain.Session {
				s, _ := domain.NewSession(domain.NewSessionID("sess-001"), "user-001", "test", "test:chat123", "", "")
				return s
			}(),
		},
	}
	sessionSvc := NewSessionApplicationService(sessionRepo, idGen)

	svc := NewRequirementDispatchService(
		reqRepo,
		projectRepo,
		agentRepo,
		sessionSvc,
		idGen,
		&mockReplicaCleanupService{},
		nil,
		&mockWorkspaceConfigForDispatch{},
		&mockWorkspaceManagerForDispatch{},
		publisher,
	)
	return svc, reqRepo
}

func TestDispatchRequirement_MaxConcurrentAgentsReached(t *testing.T) {
	svc, reqRepo := setupDispatchService(2, nil)
	ctx := context.Background()

	project, _ := domain.NewProject(domain.NewProjectID("proj-001"), "Test", "https://github.com/a/b.git", "main", []string{})
	projectRepo := newSharedMockProjectRepo()
	projectRepo.Save(ctx, project)
	svc.projectRepo = projectRepo

	requirement, _ := domain.NewRequirement(domain.NewRequirementID("req-001"), domain.NewProjectID("proj-001"), "Title", "Desc", "AC", "")
	requirement.SetRequirementType(domain.RequirementTypeNormal)
	reqRepo.requirements["req-001"] = requirement

	baseAgent, _ := domain.NewAgent(
		domain.NewAgentID("agent-001"),
		domain.NewAgentCode("agt-base-001"),
		"user-001",
		"BaseAgent",
		"desc",
		domain.AgentTypeCoding,
	)
	svc.agentRepo.(*mockAgentRepoForDispatch).agents["agt-base-001"] = baseAgent

	_, err := svc.DispatchRequirement(ctx, DispatchRequirementCommand{
		RequirementID: domain.NewRequirementID("req-001"),
		AgentCode:     "agt-base-001",
		ChannelCode:   "test",
		SessionKey:    "test:chat123",
	})

	if err != ErrMaxConcurrentAgentsReached {
		t.Errorf("期望 ErrMaxConcurrentAgentsReached, 实际 %v", err)
	}
}

func TestDispatchRequirement_MaxConcurrentAgentsAllowed(t *testing.T) {
	svc, reqRepo := setupDispatchService(1, nil)
	ctx := context.Background()

	project, _ := domain.NewProject(domain.NewProjectID("proj-001"), "Test", "https://github.com/a/b.git", "main", []string{})
	projectRepo := newSharedMockProjectRepo()
	projectRepo.Save(ctx, project)
	svc.projectRepo = projectRepo

	requirement, _ := domain.NewRequirement(domain.NewRequirementID("req-001"), domain.NewProjectID("proj-001"), "Title", "Desc", "AC", "")
	requirement.SetRequirementType(domain.RequirementTypeNormal)
	reqRepo.requirements["req-001"] = requirement

	baseAgent, _ := domain.NewAgent(
		domain.NewAgentID("agent-001"),
		domain.NewAgentCode("agt-base-001"),
		"user-001",
		"BaseAgent",
		"desc",
		domain.AgentTypeCoding,
	)
	svc.agentRepo.(*mockAgentRepoForDispatch).agents["agt-base-001"] = baseAgent

	_, err := svc.DispatchRequirement(ctx, DispatchRequirementCommand{
		RequirementID: domain.NewRequirementID("req-001"),
		AgentCode:     "agt-base-001",
		ChannelCode:   "test",
		SessionKey:    "test:chat123",
	})

	if err == ErrMaxConcurrentAgentsReached {
		t.Error("不应因并发限制被拒绝")
	}
}
