package store

import (
	"context"
	"fmt"
	"time"
)

// NotificationLog records one notification send attempt.
type NotificationLog struct {
	ID        int64
	RunID     int64
	ChannelID int64
	RuleID    *int64
	SentAt    time.Time
	Status    string
	Error     string
}

// LogNotification records a send attempt.
func (s *Store) LogNotification(ctx context.Context, in NotificationLog) error {
	sentAt := in.SentAt
	if sentAt.IsZero() {
		sentAt = s.now()
	}
	var ruleArg any
	if in.RuleID != nil {
		ruleArg = *in.RuleID
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO notifications_log(run_id, channel_id, rule_id, sent_at, status, error)
		 VALUES(?, ?, ?, ?, ?, ?)`,
		in.RunID, in.ChannelID, ruleArg, sentAt.Unix(), in.Status, in.Error)
	if err != nil {
		return fmt.Errorf("logging notification: %w", err)
	}
	return nil
}

// CountNotificationLogs returns the number of logged sends for a run.
func (s *Store) CountNotificationLogs(ctx context.Context, runID int64) (int, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notifications_log WHERE run_id=?`, runID).Scan(&n); err != nil {
		return 0, fmt.Errorf("counting notification logs: %w", err)
	}
	return n, nil
}
