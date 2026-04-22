package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type SQLiteWebhookEventTriggeredHeartbeatRepository struct {
	db *sql.DB
}

func NewSQLiteWebhookEventTriggeredHeartbeatRepository(db *sql.DB) *SQLiteWebhookEventTriggeredHeartbeatRepository {
	return &SQLiteWebhookEventTriggeredHeartbeatRepository{db: db}
}

func (r *SQLiteWebhookEventTriggeredHeartbeatRepository) Save(ctx context.Context, triggered *domain.WebhookEventTriggeredHeartbeat) error {
	snap := triggered.ToSnapshot()
	query := `
		INSERT INTO webhook_event_triggered_heartbeats (id, webhook_event_log_id, heartbeat_id, requirement_id, triggered_at)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(
		ctx,
		query,
		snap.ID.String(),
		snap.WebhookEventLogID.String(),
		snap.HeartbeatID.String(),
		snap.RequirementID,
		snap.TriggeredAt.Unix(),
	)
	return err
}

func (r *SQLiteWebhookEventTriggeredHeartbeatRepository) FindByEventLogID(ctx context.Context, eventLogID domain.WebhookEventLogID) ([]*domain.WebhookEventTriggeredHeartbeat, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, webhook_event_log_id, heartbeat_id, requirement_id, triggered_at
		FROM webhook_event_triggered_heartbeats WHERE webhook_event_log_id = ? ORDER BY triggered_at`, eventLogID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWebhookEventTriggeredHeartbeats(rows)
}

func (r *SQLiteWebhookEventTriggeredHeartbeatRepository) DeleteByEventLogID(ctx context.Context, eventLogID domain.WebhookEventLogID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM webhook_event_triggered_heartbeats WHERE webhook_event_log_id = ?`, eventLogID.String())
	return err
}

func scanWebhookEventTriggeredHeartbeats(rows *sql.Rows) ([]*domain.WebhookEventTriggeredHeartbeat, error) {
	triggered := make([]*domain.WebhookEventTriggeredHeartbeat, 0)
	for rows.Next() {
		t, err := scanWebhookEventTriggeredHeartbeat(rows)
		if err != nil {
			return nil, err
		}
		if t != nil {
			triggered = append(triggered, t)
		}
	}
	return triggered, rows.Err()
}

func scanWebhookEventTriggeredHeartbeat(scanner rowScanner) (*domain.WebhookEventTriggeredHeartbeat, error) {
	var (
		idStr              string
		eventLogIDStr      string
		heartbeatIDStr     string
		requirementID      sql.NullString
		triggeredAtUnix    int64
	)
	err := scanner.Scan(&idStr, &eventLogIDStr, &heartbeatIDStr, &requirementID, &triggeredAtUnix)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	triggered := &domain.WebhookEventTriggeredHeartbeat{}
	triggered.FromSnapshot(domain.WebhookEventTriggeredHeartbeatSnapshot{
		ID:               domain.NewWebhookEventTriggeredHeartbeatID(idStr),
		WebhookEventLogID: domain.NewWebhookEventLogID(eventLogIDStr),
		HeartbeatID:      domain.NewHeartbeatID(heartbeatIDStr),
		RequirementID:    requirementID.String,
		TriggeredAt:      time.Unix(triggeredAtUnix, 0),
	})
	return triggered, nil
}
