package domain

import "context"

type HeartbeatScenarioRepository interface {
	Save(ctx context.Context, scenario *HeartbeatScenario) error
	FindByID(ctx context.Context, id HeartbeatScenarioID) (*HeartbeatScenario, error)
	FindByCode(ctx context.Context, code string) (*HeartbeatScenario, error)
	FindAll(ctx context.Context) ([]*HeartbeatScenario, error)
	Delete(ctx context.Context, id HeartbeatScenarioID) error
}
