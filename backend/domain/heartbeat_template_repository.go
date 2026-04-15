package domain

import "context"

type HeartbeatTemplateRepository interface {
	Save(ctx context.Context, template *HeartbeatTemplate) error
	FindByID(ctx context.Context, id HeartbeatTemplateID) (*HeartbeatTemplate, error)
	FindAll(ctx context.Context) ([]*HeartbeatTemplate, error)
	Delete(ctx context.Context, id HeartbeatTemplateID) error
}
