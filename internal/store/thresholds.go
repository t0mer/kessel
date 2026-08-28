package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// ThresholdMode is how a threshold rule is evaluated.
type ThresholdMode = string

const (
	ThresholdAbsolute ThresholdMode = "absolute"
	ThresholdDelta    ThresholdMode = "delta"
)

// ThresholdRule triggers a notification when a category score crosses a bound.
type ThresholdRule struct {
	ID       int64
	SiteID   int64
	Category string
	Mode     string
	Value    float64
	Enabled  bool
}

const thresholdCols = `id, site_id, category, mode, value, enabled`

func scanThreshold(row interface{ Scan(...any) error }) (ThresholdRule, error) {
	var r ThresholdRule
	var enabled int
	if err := row.Scan(&r.ID, &r.SiteID, &r.Category, &r.Mode, &r.Value, &enabled); err != nil {
		return ThresholdRule{}, err
	}
	r.Enabled = enabled != 0
	return r, nil
}

// CreateThresholdRule inserts a new (enabled) rule.
func (s *Store) CreateThresholdRule(ctx context.Context, in ThresholdRule) (ThresholdRule, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO threshold_rules(site_id, category, mode, value, enabled) VALUES(?, ?, ?, ?, 1)`,
		in.SiteID, in.Category, in.Mode, in.Value)
	if err != nil {
		return ThresholdRule{}, fmt.Errorf("inserting threshold rule: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetThresholdRule(ctx, id)
}

// GetThresholdRule returns a rule by ID.
func (s *Store) GetThresholdRule(ctx context.Context, id int64) (ThresholdRule, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+thresholdCols+` FROM threshold_rules WHERE id=?`, id)
	r, err := scanThreshold(row)
	if errors.Is(err, sql.ErrNoRows) {
		return ThresholdRule{}, ErrNotFound
	}
	if err != nil {
		return ThresholdRule{}, fmt.Errorf("getting threshold rule: %w", err)
	}
	return r, nil
}

func (s *Store) queryThresholds(ctx context.Context, where string, args ...any) ([]ThresholdRule, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+thresholdCols+` FROM threshold_rules`+where+` ORDER BY id`, args...)
	if err != nil {
		return nil, fmt.Errorf("listing threshold rules: %w", err)
	}
	defer rows.Close()
	var out []ThresholdRule
	for rows.Next() {
		r, err := scanThreshold(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning threshold rule: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListThresholdRulesBySite returns all rules for a site.
func (s *Store) ListThresholdRulesBySite(ctx context.Context, siteID int64) ([]ThresholdRule, error) {
	return s.queryThresholds(ctx, " WHERE site_id=?", siteID)
}

// ListEnabledThresholdRulesBySite returns enabled rules for a site.
func (s *Store) ListEnabledThresholdRulesBySite(ctx context.Context, siteID int64) ([]ThresholdRule, error) {
	return s.queryThresholds(ctx, " WHERE site_id=? AND enabled=1", siteID)
}

// UpdateThresholdRule updates category, mode, value, enabled.
func (s *Store) UpdateThresholdRule(ctx context.Context, in ThresholdRule) (ThresholdRule, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE threshold_rules SET category=?, mode=?, value=?, enabled=? WHERE id=?`,
		in.Category, in.Mode, in.Value, boolToInt(in.Enabled), in.ID)
	if err != nil {
		return ThresholdRule{}, fmt.Errorf("updating threshold rule: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ThresholdRule{}, ErrNotFound
	}
	return s.GetThresholdRule(ctx, in.ID)
}

// SetThresholdRuleEnabled toggles a rule's enabled flag.
func (s *Store) SetThresholdRuleEnabled(ctx context.Context, id int64, enabled bool) error {
	res, err := s.db.ExecContext(ctx, `UPDATE threshold_rules SET enabled=? WHERE id=?`, boolToInt(enabled), id)
	if err != nil {
		return fmt.Errorf("setting threshold rule enabled: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteThresholdRule removes a rule.
func (s *Store) DeleteThresholdRule(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM threshold_rules WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("deleting threshold rule: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
