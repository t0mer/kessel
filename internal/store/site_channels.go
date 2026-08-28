package store

import (
	"context"
	"fmt"
)

// SetSiteChannels replaces the set of channels linked to a site.
func (s *Store) SetSiteChannels(ctx context.Context, siteID int64, channelIDs []int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM site_channels WHERE site_id=?`, siteID); err != nil {
		return fmt.Errorf("clearing site channels: %w", err)
	}
	for _, id := range channelIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO site_channels(site_id, channel_id) VALUES(?, ?)`, siteID, id); err != nil {
			return fmt.Errorf("linking channel %d: %w", id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// ListChannelIDsBySite returns the IDs of channels linked to a site.
func (s *Store) ListChannelIDsBySite(ctx context.Context, siteID int64) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT channel_id FROM site_channels WHERE site_id=? ORDER BY channel_id`, siteID)
	if err != nil {
		return nil, fmt.Errorf("listing site channel ids: %w", err)
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning channel id: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// ListEnabledChannelsBySite returns the enabled channels linked to a site,
// ordered by name (used by the notification dispatcher).
func (s *Store) ListEnabledChannelsBySite(ctx context.Context, siteID int64) ([]Channel, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+channelCols+` FROM channels c
		 JOIN site_channels sc ON sc.channel_id = c.id
		 WHERE sc.site_id = ? AND c.enabled = 1
		 ORDER BY c.name COLLATE NOCASE`, siteID)
	if err != nil {
		return nil, fmt.Errorf("listing enabled channels for site: %w", err)
	}
	defer rows.Close()
	var out []Channel
	for rows.Next() {
		c, err := scanChannel(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning channel: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
