package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// RunStatus is the outcome of a run.
type RunStatus = string

const (
	RunStatusSuccess RunStatus = "success"
	RunStatusError   RunStatus = "error"
)

// Run is a single PSI check result (or failure) for a site+strategy.
type Run struct {
	ID            int64
	SiteID        int64
	Strategy      string
	StartedAt     time.Time
	FinishedAt    time.Time
	Status        string
	Perf          *float64
	Accessibility *float64
	BestPractices *float64
	SEO           *float64
	LCPms         *float64
	CLS           *float64
	TBTms         *float64
	FCPms         *float64
	SIms          *float64
	TTIms         *float64
	RawJSONGz     []byte
	ReportPath    string
	Error         string
}

// argF turns a nullable float into an insert arg (nil -> SQL NULL).
func argF(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

func ptrF(nf sql.NullFloat64) *float64 {
	if !nf.Valid {
		return nil
	}
	x := nf.Float64
	return &x
}

const runCols = `id, site_id, strategy, started_at, finished_at, status,
	perf, accessibility, best_practices, seo,
	lcp_ms, cls, tbt_ms, fcp_ms, si_ms, tti_ms,
	raw_json_gz, report_path, error`

// CreateRun inserts a run and returns the stored row.
func (s *Store) CreateRun(ctx context.Context, in Run) (Run, error) {
	res, err := s.conn().ExecContext(ctx,
		`INSERT INTO runs(site_id, strategy, started_at, finished_at, status,
			perf, accessibility, best_practices, seo,
			lcp_ms, cls, tbt_ms, fcp_ms, si_ms, tti_ms,
			raw_json_gz, report_path, error)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		in.SiteID, in.Strategy, in.StartedAt.Unix(), in.FinishedAt.Unix(), in.Status,
		argF(in.Perf), argF(in.Accessibility), argF(in.BestPractices), argF(in.SEO),
		argF(in.LCPms), argF(in.CLS), argF(in.TBTms), argF(in.FCPms), argF(in.SIms), argF(in.TTIms),
		in.RawJSONGz, in.ReportPath, in.Error)
	if err != nil {
		return Run{}, fmt.Errorf("inserting run: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetRun(ctx, id)
}

func scanRun(row interface{ Scan(...any) error }) (Run, error) {
	var r Run
	var started, finished int64
	var perf, acc, bp, seo, lcp, cls, tbt, fcp, si, tti sql.NullFloat64
	var reportPath, errStr sql.NullString
	var raw []byte
	if err := row.Scan(
		&r.ID, &r.SiteID, &r.Strategy, &started, &finished, &r.Status,
		&perf, &acc, &bp, &seo,
		&lcp, &cls, &tbt, &fcp, &si, &tti,
		&raw, &reportPath, &errStr,
	); err != nil {
		return Run{}, err
	}
	r.StartedAt = time.Unix(started, 0)
	r.FinishedAt = time.Unix(finished, 0)
	r.Perf, r.Accessibility, r.BestPractices, r.SEO = ptrF(perf), ptrF(acc), ptrF(bp), ptrF(seo)
	r.LCPms, r.CLS, r.TBTms, r.FCPms, r.SIms, r.TTIms = ptrF(lcp), ptrF(cls), ptrF(tbt), ptrF(fcp), ptrF(si), ptrF(tti)
	r.RawJSONGz = raw
	r.ReportPath = reportPath.String
	r.Error = errStr.String
	return r, nil
}

// GetRun returns a run by ID.
func (s *Store) GetRun(ctx context.Context, id int64) (Run, error) {
	row := s.conn().QueryRowContext(ctx, `SELECT `+runCols+` FROM runs WHERE id=?`, id)
	r, err := scanRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Run{}, ErrNotFound
	}
	if err != nil {
		return Run{}, fmt.Errorf("getting run: %w", err)
	}
	return r, nil
}

// RunFilter selects and paginates runs.
type RunFilter struct {
	SiteID   *int64
	Strategy string
	Limit    int
	Offset   int
}

func (f RunFilter) where() (string, []any) {
	var conds []string
	var args []any
	if f.SiteID != nil {
		conds = append(conds, "site_id=?")
		args = append(args, *f.SiteID)
	}
	if f.Strategy != "" {
		conds = append(conds, "strategy=?")
		args = append(args, f.Strategy)
	}
	if len(conds) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// ListRuns returns runs matching the filter, newest first.
func (s *Store) ListRuns(ctx context.Context, f RunFilter) ([]Run, error) {
	where, args := f.where()
	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	args = append(args, limit, f.Offset)
	rows, err := s.conn().QueryContext(ctx,
		`SELECT `+runCols+` FROM runs`+where+
			` ORDER BY started_at DESC, id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, fmt.Errorf("listing runs: %w", err)
	}
	defer rows.Close()
	var out []Run
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning run: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CountRuns returns the total number of runs matching the filter.
func (s *Store) CountRuns(ctx context.Context, f RunFilter) (int, error) {
	where, args := f.where()
	var n int
	if err := s.conn().QueryRowContext(ctx, `SELECT COUNT(*) FROM runs`+where, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("counting runs: %w", err)
	}
	return n, nil
}

// LatestRun returns the most recent run for a site+strategy.
func (s *Store) LatestRun(ctx context.Context, siteID int64, strategy string) (Run, error) {
	row := s.conn().QueryRowContext(ctx,
		`SELECT `+runCols+` FROM runs WHERE site_id=? AND strategy=?
		 ORDER BY started_at DESC, id DESC LIMIT 1`, siteID, strategy)
	r, err := scanRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Run{}, ErrNotFound
	}
	if err != nil {
		return Run{}, fmt.Errorf("getting latest run: %w", err)
	}
	return r, nil
}

// SetRunReportPath records the rendered report path for a run.
func (s *Store) SetRunReportPath(ctx context.Context, runID int64, path string) error {
	res, err := s.conn().ExecContext(ctx, `UPDATE runs SET report_path=? WHERE id=?`, path, runID)
	if err != nil {
		return fmt.Errorf("setting run report path: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// PreviousSuccessfulRun returns the newest successful run for a site+strategy
// started strictly before beforeStartedAt.
func (s *Store) PreviousSuccessfulRun(ctx context.Context, siteID int64, strategy string, beforeStartedAt time.Time) (Run, error) {
	row := s.conn().QueryRowContext(ctx,
		`SELECT `+runCols+` FROM runs
		 WHERE site_id=? AND strategy=? AND status=? AND started_at < ?
		 ORDER BY started_at DESC, id DESC LIMIT 1`,
		siteID, strategy, RunStatusSuccess, beforeStartedAt.Unix())
	r, err := scanRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Run{}, ErrNotFound
	}
	if err != nil {
		return Run{}, fmt.Errorf("getting previous successful run: %w", err)
	}
	return r, nil
}
