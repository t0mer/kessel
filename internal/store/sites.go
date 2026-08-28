package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrNotFound is returned when a requested row does not exist.
var ErrNotFound = errors.New("not found")

// Strategy is the PSI strategy for a site.
type Strategy = string

const (
	StrategyMobile  Strategy = "mobile"
	StrategyDesktop Strategy = "desktop"
	StrategyBoth    Strategy = "both"
)

// Site is a monitored website.
type Site struct {
	ID        int64
	Name      string
	Slug      string
	URL       string
	Strategy  string
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CreateSite inserts a new site, assigning a unique slug and timestamps.
// New sites are always created enabled; use UpdateSite/SetSiteEnabled to disable.
func (s *Store) CreateSite(ctx context.Context, in Site) (Site, error) {
	base := in.Slug
	if base == "" {
		base = in.Name
	}
	base = Slugify(base)
	slug, err := s.uniqueSlug(ctx, base)
	if err != nil {
		return Site{}, err
	}
	now := s.now().Unix()
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO sites(name, slug, url, strategy, enabled, created_at, updated_at)
		 VALUES(?, ?, ?, ?, 1, ?, ?)`,
		in.Name, slug, in.URL, in.Strategy, now, now)
	if err != nil {
		return Site{}, fmt.Errorf("inserting site: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetSite(ctx, id)
}

func (s *Store) uniqueSlug(ctx context.Context, base string) (string, error) {
	candidate := base
	for i := 2; ; i++ {
		var n int
		if err := s.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM sites WHERE slug=?`, candidate).Scan(&n); err != nil {
			return "", fmt.Errorf("checking slug uniqueness: %w", err)
		}
		if n == 0 {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}

func scanSite(row interface{ Scan(...any) error }) (Site, error) {
	var st Site
	var enabled int
	var created, updated int64
	if err := row.Scan(&st.ID, &st.Name, &st.Slug, &st.URL, &st.Strategy, &enabled, &created, &updated); err != nil {
		return Site{}, err
	}
	st.Enabled = enabled != 0
	st.CreatedAt = time.Unix(created, 0)
	st.UpdatedAt = time.Unix(updated, 0)
	return st, nil
}

const siteCols = `id, name, slug, url, strategy, enabled, created_at, updated_at`

// GetSite returns a site by ID.
func (s *Store) GetSite(ctx context.Context, id int64) (Site, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+siteCols+` FROM sites WHERE id=?`, id)
	st, err := scanSite(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Site{}, ErrNotFound
	}
	if err != nil {
		return Site{}, fmt.Errorf("getting site: %w", err)
	}
	return st, nil
}

// GetSiteBySlug returns a site by slug.
func (s *Store) GetSiteBySlug(ctx context.Context, slug string) (Site, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+siteCols+` FROM sites WHERE slug=?`, slug)
	st, err := scanSite(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Site{}, ErrNotFound
	}
	if err != nil {
		return Site{}, fmt.Errorf("getting site by slug: %w", err)
	}
	return st, nil
}

// ListSites returns all sites ordered by name.
func (s *Store) ListSites(ctx context.Context) ([]Site, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+siteCols+` FROM sites ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("listing sites: %w", err)
	}
	defer rows.Close()
	var out []Site
	for rows.Next() {
		st, err := scanSite(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning site: %w", err)
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

// UpdateSite updates mutable fields (name, url, strategy, enabled). Slug is immutable.
func (s *Store) UpdateSite(ctx context.Context, in Site) (Site, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE sites SET name=?, url=?, strategy=?, enabled=?, updated_at=? WHERE id=?`,
		in.Name, in.URL, in.Strategy, boolToInt(in.Enabled), s.now().Unix(), in.ID)
	if err != nil {
		return Site{}, fmt.Errorf("updating site: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Site{}, ErrNotFound
	}
	return s.GetSite(ctx, in.ID)
}

// SetSiteEnabled toggles a site's enabled flag.
func (s *Store) SetSiteEnabled(ctx context.Context, id int64, enabled bool) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE sites SET enabled=?, updated_at=? WHERE id=?`,
		boolToInt(enabled), s.now().Unix(), id)
	if err != nil {
		return fmt.Errorf("setting site enabled: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteSite removes a site; schedules, rules, and runs cascade via FK.
func (s *Store) DeleteSite(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM sites WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("deleting site: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
