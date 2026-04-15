package domain

import "context"

type HeartbeatRepository interface {
	Save(ctx context.Context, hb *Heartbeat) error
	FindByID(ctx context.Context, id HeartbeatID) (*Heartbeat, error)
	FindByProjectID(ctx context.Context, projectID ProjectID) ([]*Heartbeat, error)
	FindAllEnabled(ctx context.Context) ([]*Heartbeat, error)
	Delete(ctx context.Context, id HeartbeatID) error
}
