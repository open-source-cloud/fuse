package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// WorkflowNotification represents a parsed PG NOTIFY payload from the workflow_state_change channel.
type WorkflowNotification struct {
	WorkflowID string
	State      string
	ClaimedBy  string
}

// PgListener subscribes to PostgreSQL LISTEN/NOTIFY channels using a dedicated connection.
type PgListener struct {
	conn *pgx.Conn
}

// NewPgListener creates a new PgListener from the given DSN.
// Uses a dedicated connection (not from the pool) because LISTEN requires
// a persistent connection that is not recycled.
func NewPgListener(ctx context.Context, dsn string) (*PgListener, error) {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres/listener: connect: %w", err)
	}
	return &PgListener{conn: conn}, nil
}

// Listen subscribes to the workflow_state_change channel and sends parsed
// notifications to the returned channel. Blocks until ctx is cancelled.
// The caller should run this in a goroutine.
func (l *PgListener) Listen(ctx context.Context, ch chan<- WorkflowNotification) error {
	_, err := l.conn.Exec(ctx, "LISTEN workflow_state_change")
	if err != nil {
		return fmt.Errorf("postgres/listener: LISTEN: %w", err)
	}
	log.Info().Msg("postgres/listener: subscribed to workflow_state_change")

	for {
		notification, waitErr := l.conn.WaitForNotification(ctx)
		if waitErr != nil {
			if ctx.Err() != nil {
				return nil // context cancelled, clean shutdown
			}
			return fmt.Errorf("postgres/listener: wait: %w", waitErr)
		}

		wn := parseNotificationPayload(notification.Payload)
		select {
		case ch <- wn:
		case <-ctx.Done():
			return nil
		}
	}
}

// Close closes the dedicated listener connection.
func (l *PgListener) Close(ctx context.Context) error {
	return l.conn.Close(ctx)
}

// parseNotificationPayload parses "workflow_id:state:claimed_by" into a WorkflowNotification.
func parseNotificationPayload(payload string) WorkflowNotification {
	parts := strings.SplitN(payload, ":", 3)
	wn := WorkflowNotification{}
	if len(parts) >= 1 {
		wn.WorkflowID = parts[0]
	}
	if len(parts) >= 2 {
		wn.State = parts[1]
	}
	if len(parts) >= 3 {
		wn.ClaimedBy = parts[2]
	}
	return wn
}
