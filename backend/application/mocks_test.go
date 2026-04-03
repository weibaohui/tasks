package application

import (
	"context"

	"github.com/weibh/taskmanager/domain"
)

// sharedMockProjectRepo is a shared mock implementation of domain.ProjectRepository
// for use in tests across multiple test files.
type sharedMockProjectRepo struct {
	projects map[string]*domain.Project
}

func newSharedMockProjectRepo() *sharedMockProjectRepo {
	return &sharedMockProjectRepo{
		projects: make(map[string]*domain.Project),
	}
}

func (m *sharedMockProjectRepo) Save(ctx context.Context, project *domain.Project) error {
	m.projects[project.ID().String()] = project
	return nil
}

func (m *sharedMockProjectRepo) FindByID(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
	project, ok := m.projects[id.String()]
	if !ok {
		return nil, nil
	}
	return project, nil
}

func (m *sharedMockProjectRepo) FindAll(ctx context.Context) ([]*domain.Project, error) {
	var result []*domain.Project
	for _, project := range m.projects {
		result = append(result, project)
	}
	return result, nil
}

func (m *sharedMockProjectRepo) Delete(ctx context.Context, id domain.ProjectID) error {
	delete(m.projects, id.String())
	return nil
}
