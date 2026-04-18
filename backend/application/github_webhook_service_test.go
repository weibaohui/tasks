package application

import (
	"context"
	"errors"
	"testing"

	"github.com/weibh/taskmanager/domain"
)

type testMockHeartbeatRepo struct {
	heartbeats []*domain.Heartbeat
	findErr    error
}

func (m *testMockHeartbeatRepo) FindByID(ctx context.Context, id domain.HeartbeatID) (*domain.Heartbeat, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	for _, hb := range m.heartbeats {
		if hb.ID().String() == id.String() {
			return hb, nil
		}
	}
	return nil, nil
}

func (m *testMockHeartbeatRepo) FindByProjectID(ctx context.Context, projectID domain.ProjectID) ([]*domain.Heartbeat, error) {
	var result []*domain.Heartbeat
	for _, hb := range m.heartbeats {
		if hb.ProjectID().String() == projectID.String() {
			result = append(result, hb)
		}
	}
	return result, nil
}

func (m *testMockHeartbeatRepo) Save(ctx context.Context, hb *domain.Heartbeat) error    { return nil }
func (m *testMockHeartbeatRepo) Delete(ctx context.Context, id domain.HeartbeatID) error { return nil }
func (m *testMockHeartbeatRepo) FindAll(ctx context.Context) ([]*domain.Heartbeat, error) {
	return m.heartbeats, nil
}
func (m *testMockHeartbeatRepo) FindAllEnabled(ctx context.Context) ([]*domain.Heartbeat, error) {
	return m.heartbeats, nil
}

type testMockGitHubWebhookConfigRepo struct {
	configs []*domain.GitHubWebhookConfig
}

func (m *testMockGitHubWebhookConfigRepo) Save(ctx context.Context, config *domain.GitHubWebhookConfig) error {
	m.configs = append(m.configs, config)
	return nil
}

func (m *testMockGitHubWebhookConfigRepo) FindByID(ctx context.Context, id domain.GitHubWebhookConfigID) (*domain.GitHubWebhookConfig, error) {
	for _, c := range m.configs {
		if c.ID().String() == id.String() {
			return c, nil
		}
	}
	return nil, nil
}

func (m *testMockGitHubWebhookConfigRepo) FindByProjectID(ctx context.Context, projectID domain.ProjectID) (*domain.GitHubWebhookConfig, error) {
	for _, c := range m.configs {
		if c.ProjectID().String() == projectID.String() {
			return c, nil
		}
	}
	return nil, nil
}

func (m *testMockGitHubWebhookConfigRepo) FindAll(ctx context.Context) ([]*domain.GitHubWebhookConfig, error) {
	return m.configs, nil
}

func (m *testMockGitHubWebhookConfigRepo) FindAllEnabled(ctx context.Context) ([]*domain.GitHubWebhookConfig, error) {
	return m.configs, nil
}

func (m *testMockGitHubWebhookConfigRepo) Delete(ctx context.Context, id domain.GitHubWebhookConfigID) error {
	return nil
}

type testMockWebhookEventLogRepo struct {
	logs []*domain.WebhookEventLog
}

func (m *testMockWebhookEventLogRepo) Save(ctx context.Context, log *domain.WebhookEventLog) error {
	m.logs = append(m.logs, log)
	return nil
}

func (m *testMockWebhookEventLogRepo) FindByID(ctx context.Context, id domain.WebhookEventLogID) (*domain.WebhookEventLog, error) {
	for _, l := range m.logs {
		if l.ID().String() == id.String() {
			return l, nil
		}
	}
	return nil, nil
}

func (m *testMockWebhookEventLogRepo) FindByProjectID(ctx context.Context, projectID domain.ProjectID, limit, offset int) ([]*domain.WebhookEventLog, error) {
	return m.logs, nil
}

func (m *testMockWebhookEventLogRepo) CountByProjectID(ctx context.Context, projectID domain.ProjectID) (int, error) {
	return len(m.logs), nil
}

func (m *testMockWebhookEventLogRepo) DeleteByProjectID(ctx context.Context, projectID domain.ProjectID) error {
	m.logs = nil
	return nil
}

func (m *testMockWebhookEventLogRepo) Delete(ctx context.Context, id domain.WebhookEventLogID) error {
	return nil
}

type testMockWebhookBindingRepo struct {
	bindings []*domain.WebhookHeartbeatBinding
}

func (m *testMockWebhookBindingRepo) Save(ctx context.Context, binding *domain.WebhookHeartbeatBinding) error {
	m.bindings = append(m.bindings, binding)
	return nil
}

func (m *testMockWebhookBindingRepo) FindByID(ctx context.Context, id domain.WebhookHeartbeatBindingID) (*domain.WebhookHeartbeatBinding, error) {
	for _, b := range m.bindings {
		if b.ID().String() == id.String() {
			return b, nil
		}
	}
	return nil, nil
}

func (m *testMockWebhookBindingRepo) FindByConfigID(ctx context.Context, configID domain.GitHubWebhookConfigID) ([]*domain.WebhookHeartbeatBinding, error) {
	var result []*domain.WebhookHeartbeatBinding
	for _, b := range m.bindings {
		if b.ConfigID().String() == configID.String() {
			result = append(result, b)
		}
	}
	return result, nil
}

func (m *testMockWebhookBindingRepo) FindByConfigIDAndEventType(ctx context.Context, configID domain.GitHubWebhookConfigID, eventType string) ([]*domain.WebhookHeartbeatBinding, error) {
	var result []*domain.WebhookHeartbeatBinding
	for _, b := range m.bindings {
		if b.ConfigID().String() == configID.String() && b.GitHubEventType() == eventType {
			result = append(result, b)
		}
	}
	return result, nil
}

func (m *testMockWebhookBindingRepo) Delete(ctx context.Context, id domain.WebhookHeartbeatBindingID) error {
	return nil
}

type testMockIDGen struct {
	id string
}

func (m *testMockIDGen) Generate() string {
	return m.id
}

func TestGitHubWebhookService_CreateBinding_HeartbeatNotFound(t *testing.T) {
	ctx := context.Background()

	projectID := "proj-001"
	configID := "config-001"
	eventType := "push"
	heartbeatID := "hb-nonexistent"

	configRepo := &testMockGitHubWebhookConfigRepo{}
	eventLogRepo := &testMockWebhookEventLogRepo{}
	bindingRepo := &testMockWebhookBindingRepo{}
	heartbeatRepo := &testMockHeartbeatRepo{heartbeats: []*domain.Heartbeat{}}
	idGen := &testMockIDGen{id: "binding-001"}

	triggerService := &HeartbeatTriggerService{}

	svc := NewGitHubWebhookService(
		configRepo,
		eventLogRepo,
		bindingRepo,
		heartbeatRepo,
		triggerService,
		idGen,
	)

	_, err := svc.CreateBinding(ctx, projectID, configID, eventType, heartbeatID)
	if err == nil {
		t.Error("expected error for non-existent heartbeat")
	}
}

func TestGitHubWebhookService_CreateBinding_HeartbeatExists(t *testing.T) {
	ctx := context.Background()

	project, _ := domain.NewProject(
		domain.NewProjectID("proj-001"),
		"Test Project",
		"git@github.com:test/repo.git",
		"main",
		nil,
	)

	hb, _ := domain.NewHeartbeat(
		domain.NewHeartbeatID("hb-001"),
		project.ID(),
		"Test Heartbeat",
		5,
		"heartbeat content",
		"agent-001",
		"heartbeat",
	)

	projectID := "proj-001"
	configID := "config-001"
	eventType := "push"
	heartbeatID := "hb-001"

	configRepo := &testMockGitHubWebhookConfigRepo{}
	eventLogRepo := &testMockWebhookEventLogRepo{}
	bindingRepo := &testMockWebhookBindingRepo{}
	heartbeatRepo := &testMockHeartbeatRepo{heartbeats: []*domain.Heartbeat{hb}}
	idGen := &testMockIDGen{id: "binding-001"}

	triggerService := &HeartbeatTriggerService{}

	svc := NewGitHubWebhookService(
		configRepo,
		eventLogRepo,
		bindingRepo,
		heartbeatRepo,
		triggerService,
		idGen,
	)

	binding, err := svc.CreateBinding(ctx, projectID, configID, eventType, heartbeatID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if binding == nil {
		t.Error("binding should not be nil")
	}
	if binding.HeartbeatID().String() != heartbeatID {
		t.Errorf("expected heartbeat ID %s, got %s", heartbeatID, binding.HeartbeatID().String())
	}
}

func TestGitHubWebhookService_CreateBinding_HeartbeatRepoError(t *testing.T) {
	ctx := context.Background()

	projectID := "proj-001"
	configID := "config-001"
	eventType := "push"
	heartbeatID := "hb-001"

	configRepo := &testMockGitHubWebhookConfigRepo{}
	eventLogRepo := &testMockWebhookEventLogRepo{}
	bindingRepo := &testMockWebhookBindingRepo{}
	heartbeatRepo := &testMockHeartbeatRepo{findErr: errors.New("database error")}
	idGen := &testMockIDGen{id: "binding-001"}

	triggerService := &HeartbeatTriggerService{}

	svc := NewGitHubWebhookService(
		configRepo,
		eventLogRepo,
		bindingRepo,
		heartbeatRepo,
		triggerService,
		idGen,
	)

	_, err := svc.CreateBinding(ctx, projectID, configID, eventType, heartbeatID)
	if err == nil {
		t.Error("expected error from heartbeat repo")
	}
}
