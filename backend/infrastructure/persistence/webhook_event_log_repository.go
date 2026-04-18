package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/weibh/taskmanager/domain"
)

type SQLiteWebhookEventLogRepository struct {
	db *sql.DB
}

func NewSQLiteWebhookEventLogRepository(db *sql.DB) *SQLiteWebhookEventLogRepository {
	return &SQLiteWebhookEventLogRepository{db: db}
}

func (r *SQLiteWebhookEventLogRepository) Save(ctx context.Context, log *domain.WebhookEventLog) error {
	snap := log.ToSnapshot()
	query := `
		INSERT INTO webhook_event_logs (id, project_id, event_type, payload, forwarder_status, trigger_heartbeat_id, error_message, received_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			forwarder_status=excluded.forwarder_status,
			trigger_heartbeat_id=excluded.trigger_heartbeat_id,
			error_message=excluded.error_message
	`
	_, err := r.db.ExecContext(
		ctx,
		query,
		snap.ID.String(),
		snap.ProjectID.String(),
		snap.EventType,
		snap.Payload,
		string(snap.Status),
		snap.TriggerHeartbeatID,
		snap.ErrorMessage,
		snap.ReceivedAt.Unix(),
	)
	return err
}

func (r *SQLiteWebhookEventLogRepository) FindByID(ctx context.Context, id domain.WebhookEventLogID) (*domain.WebhookEventLog, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, project_id, event_type, payload, forwarder_status, trigger_heartbeat_id, error_message, received_at
		FROM webhook_event_logs WHERE id = ?`, id.String())
	return scanWebhookEventLog(row)
}

func (r *SQLiteWebhookEventLogRepository) FindByProjectID(ctx context.Context, projectID domain.ProjectID, limit, offset int) ([]*domain.WebhookEventLog, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, event_type, payload, forwarder_status, trigger_heartbeat_id, error_message, received_at
		FROM webhook_event_logs WHERE project_id = ? ORDER BY received_at DESC LIMIT ? OFFSET ?`, projectID.String(), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWebhookEventLogs(rows)
}

func (r *SQLiteWebhookEventLogRepository) CountByProjectID(ctx context.Context, projectID domain.ProjectID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM webhook_event_logs WHERE project_id = ?`, projectID.String()).Scan(&count)
	return count, err
}

func (r *SQLiteWebhookEventLogRepository) DeleteByProjectID(ctx context.Context, projectID domain.ProjectID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM webhook_event_logs WHERE project_id = ?`, projectID.String())
	return err
}

func (r *SQLiteWebhookEventLogRepository) Delete(ctx context.Context, id domain.WebhookEventLogID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM webhook_event_logs WHERE id = ?`, id.String())
	return err
}

func scanWebhookEventLogs(rows *sql.Rows) ([]*domain.WebhookEventLog, error) {
	logs := make([]*domain.WebhookEventLog, 0)
	for rows.Next() {
		log, err := scanWebhookEventLog(rows)
		if err != nil {
			return nil, err
		}
		if log != nil {
			logs = append(logs, log)
		}
	}
	return logs, rows.Err()
}

func scanWebhookEventLog(scanner rowScanner) (*domain.WebhookEventLog, error) {
	var (
		idStr              string
		projectIDStr       string
		eventType          string
		payload            string
		status             string
		triggerHeartbeatID sql.NullString
		errorMessage       sql.NullString
		receivedAtUnix     int64
	)
	err := scanner.Scan(&idStr, &projectIDStr, &eventType, &payload, &status, &triggerHeartbeatID, &errorMessage, &receivedAtUnix)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	log := &domain.WebhookEventLog{}
	log.FromSnapshot(domain.WebhookEventLogSnapshot{
		ID:                 domain.NewWebhookEventLogID(idStr),
		ProjectID:          domain.NewProjectID(projectIDStr),
		EventType:          eventType,
		Payload:            payload,
		Status:             domain.WebhookEventStatus(status),
		TriggerHeartbeatID: triggerHeartbeatID.String,
		ErrorMessage:       errorMessage.String,
		ReceivedAt:         time.Unix(receivedAtUnix, 0),
	})
	return log, nil
}
