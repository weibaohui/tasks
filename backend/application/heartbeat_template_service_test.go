package application

import (
	"context"
	"testing"

	"github.com/weibh/taskmanager/domain"
)

func TestHeartbeatTemplateService_CreateAndList(t *testing.T) {
	repo := NewInMemoryHeartbeatTemplateRepository()
	svc := NewHeartbeatTemplateApplicationService(repo, &mockIDGenerator{})
	ctx := context.Background()

	template, err := svc.CreateHeartbeatTemplate(ctx, CreateHeartbeatTemplateCommand{
		Name:            "PR检查模板",
		MDContent:       "检查PR",
		RequirementType: "pr_review",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if template.Name() != "PR检查模板" {
		t.Errorf("Name = %v, want PR检查模板", template.Name())
	}

	list, err := svc.ListHeartbeatTemplates(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("期望1条模板，实际 %d", len(list))
	}
	if list[0].Name() != "PR检查模板" {
		t.Errorf("列表名称不匹配")
	}
}

func TestHeartbeatTemplateService_Delete(t *testing.T) {
	repo := NewInMemoryHeartbeatTemplateRepository()
	svc := NewHeartbeatTemplateApplicationService(repo, &mockIDGenerator{})
	ctx := context.Background()

	template, _ := svc.CreateHeartbeatTemplate(ctx, CreateHeartbeatTemplateCommand{
		Name: "待删除",
	})

	if err := svc.DeleteHeartbeatTemplate(ctx, template.ID().String()); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	list, _ := svc.ListHeartbeatTemplates(ctx)
	if len(list) != 0 {
		t.Errorf("删除后期望0条，实际 %d", len(list))
	}
}

// In-memory repository for testing

type InMemoryHeartbeatTemplateRepository struct {
	data map[string]*domain.HeartbeatTemplate
}

func NewInMemoryHeartbeatTemplateRepository() *InMemoryHeartbeatTemplateRepository {
	return &InMemoryHeartbeatTemplateRepository{data: make(map[string]*domain.HeartbeatTemplate)}
}

func (r *InMemoryHeartbeatTemplateRepository) Save(_ context.Context, t *domain.HeartbeatTemplate) error {
	r.data[t.ID().String()] = t
	return nil
}

func (r *InMemoryHeartbeatTemplateRepository) FindByID(_ context.Context, id domain.HeartbeatTemplateID) (*domain.HeartbeatTemplate, error) {
	return r.data[id.String()], nil
}

func (r *InMemoryHeartbeatTemplateRepository) FindAll(_ context.Context) ([]*domain.HeartbeatTemplate, error) {
	result := make([]*domain.HeartbeatTemplate, 0, len(r.data))
	for _, v := range r.data {
		result = append(result, v)
	}
	return result, nil
}

func (r *InMemoryHeartbeatTemplateRepository) Delete(_ context.Context, id domain.HeartbeatTemplateID) error {
	delete(r.data, id.String())
	return nil
}
