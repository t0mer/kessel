package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Schedule is one cron schedule attached to a site.
type Schedule struct {
	ID        int64
	SiteID    int64
	CronExpr  string
	Enabled   bool
	CreatedAt time.Time
}

const scheduleCols = `id, site_id, cron_expr, enabled, created_at`

func scanSchedule(row interface{ Scan(...any) error }) (Schedule, error) {
	var sc Schedule
	var enabled int
	var created int64
	if err := row.Scan(&sc.ID, &sc.SiteID, &sc.CronExpr, &enabled, &created); err != nil {
		return Schedule{}, err
	}
	sc.Enabled = enabled != 0
	sc.CreatedAt = time.Unix(created, 0)
	return sc, nil
}

// CreateSchedule inserts a new (enabled) schedule for a site.
func (s *Store) CreateSchedule(ctx context.Context, in Schedule) (Schedule, error) {
	res, err := s.conn().ExecContext(ctx,
		`INSERT INTO schedules(site_id, cron_expr, enabled, created_at) VALUES(?, ?, 1, ?)`,
		in.SiteID, in.CronExpr, s.now().Unix())
	if err != nil {
		return Schedule{}, fmt.Errorf("inserting schedule: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetSchedule(ctx, id)
}

// GetSchedule returns a schedule by ID.
func (s *Store) GetSchedule(ctx context.Context, id int64) (Schedule, error) {
	row := s.conn().QueryRowContext(ctx, `SELECT `+scheduleCols+` FROM schedules WHERE id=?`, id)
	sc, err := scanSchedule(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Schedule{}, ErrNotFound
	}
	if err != nil {
		return Schedule{}, fmt.Errorf("getting schedule: %w", err)
	}
	return sc, nil
}

// ListSchedulesBySite returns all schedules for a site, oldest first.
func (s *Store) ListSchedulesBySite(ctx context.Context, siteID int64) ([]Schedule, error) {
	rows, err := s.conn().QueryContext(ctx,
		`SELECT `+scheduleCols+` FROM schedules WHERE site_id=? ORDER BY id`, siteID)
	if err != nil {
		return nil, fmt.Errorf("listing schedules: %w", err)
	}
	defer rows.Close()
	return collectSchedules(rows)
}

// ListEnabledSchedules returns every enabled schedule (for the scheduler).
func (s *Store) ListEnabledSchedules(ctx context.Context) ([]Schedule, error) {
	rows, err := s.conn().QueryContext(ctx,
		`SELECT `+scheduleCols+` FROM schedules WHERE enabled=1 ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("listing enabled schedules: %w", err)
	}
	defer rows.Close()
	return collectSchedules(rows)
}

func collectSchedules(rows *sql.Rows) ([]Schedule, error) {
	var out []Schedule
	for rows.Next() {
		sc, err := scanSchedule(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning schedule: %w", err)
		}
		out = append(out, sc)
	}
	return out, rows.Err()
}

// UpdateSchedule updates cron_expr and enabled for a schedule.
func (s *Store) UpdateSchedule(ctx context.Context, in Schedule) (Schedule, error) {
	res, err := s.conn().ExecContext(ctx,
		`UPDATE schedules SET cron_expr=?, enabled=? WHERE id=?`,
		in.CronExpr, boolToInt(in.Enabled), in.ID)
	if err != nil {
		return Schedule{}, fmt.Errorf("updating schedule: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Schedule{}, ErrNotFound
	}
	return s.GetSchedule(ctx, in.ID)
}

// SetScheduleEnabled toggles a schedule's enabled flag.
func (s *Store) SetScheduleEnabled(ctx context.Context, id int64, enabled bool) error {
	res, err := s.conn().ExecContext(ctx,
		`UPDATE schedules SET enabled=? WHERE id=?`, boolToInt(enabled), id)
	if err != nil {
		return fmt.Errorf("setting schedule enabled: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteSchedule removes a schedule.
func (s *Store) DeleteSchedule(ctx context.Context, id int64) error {
	res, err := s.conn().ExecContext(ctx, `DELETE FROM schedules WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("deleting schedule: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
