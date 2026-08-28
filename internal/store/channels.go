package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// ChannelType is a notification provider.
type ChannelType = string

const (
	ChannelShoutrrr    ChannelType = "shoutrrr"
	ChannelGreenAPI    ChannelType = "greenapi"
	ChannelWhatsAppWeb ChannelType = "whatsapp_web"
)

// Channel is a configured notification target. ConfigEncrypted holds the
// provider config (encrypted by the service layer; opaque to the store).
type Channel struct {
	ID              int64
	Type            string
	Name            string
	ConfigEncrypted []byte
	Enabled         bool
	NotifyOnSuccess bool
	NotifyOnFailure bool
}

const channelCols = `id, type, name, config_encrypted, enabled, notify_on_success, notify_on_failure`

func scanChannel(row interface{ Scan(...any) error }) (Channel, error) {
	var c Channel
	var enabled, onSucc, onFail int
	if err := row.Scan(&c.ID, &c.Type, &c.Name, &c.ConfigEncrypted, &enabled, &onSucc, &onFail); err != nil {
		return Channel{}, err
	}
	c.Enabled = enabled != 0
	c.NotifyOnSuccess = onSucc != 0
	c.NotifyOnFailure = onFail != 0
	return c, nil
}

// CreateChannel inserts a new (enabled) channel.
func (s *Store) CreateChannel(ctx context.Context, in Channel) (Channel, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO channels(type, name, config_encrypted, enabled, notify_on_success, notify_on_failure)
		 VALUES(?, ?, ?, 1, ?, ?)`,
		in.Type, in.Name, in.ConfigEncrypted, boolToInt(in.NotifyOnSuccess), boolToInt(in.NotifyOnFailure))
	if err != nil {
		return Channel{}, fmt.Errorf("inserting channel: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetChannel(ctx, id)
}

// GetChannel returns a channel by ID.
func (s *Store) GetChannel(ctx context.Context, id int64) (Channel, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+channelCols+` FROM channels WHERE id=?`, id)
	c, err := scanChannel(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Channel{}, ErrNotFound
	}
	if err != nil {
		return Channel{}, fmt.Errorf("getting channel: %w", err)
	}
	return c, nil
}

func (s *Store) queryChannels(ctx context.Context, where string) ([]Channel, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+channelCols+` FROM channels`+where+` ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("listing channels: %w", err)
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

// ListChannels returns all channels ordered by name.
func (s *Store) ListChannels(ctx context.Context) ([]Channel, error) {
	return s.queryChannels(ctx, "")
}

// ListEnabledChannels returns enabled channels ordered by name.
func (s *Store) ListEnabledChannels(ctx context.Context) ([]Channel, error) {
	return s.queryChannels(ctx, " WHERE enabled=1")
}

// UpdateChannel updates name, config, enabled, and notify flags. Type is immutable.
func (s *Store) UpdateChannel(ctx context.Context, in Channel) (Channel, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE channels SET name=?, config_encrypted=?, enabled=?, notify_on_success=?, notify_on_failure=? WHERE id=?`,
		in.Name, in.ConfigEncrypted, boolToInt(in.Enabled), boolToInt(in.NotifyOnSuccess), boolToInt(in.NotifyOnFailure), in.ID)
	if err != nil {
		return Channel{}, fmt.Errorf("updating channel: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Channel{}, ErrNotFound
	}
	return s.GetChannel(ctx, in.ID)
}

// SetChannelEnabled toggles a channel's enabled flag.
func (s *Store) SetChannelEnabled(ctx context.Context, id int64, enabled bool) error {
	res, err := s.db.ExecContext(ctx, `UPDATE channels SET enabled=? WHERE id=?`, boolToInt(enabled), id)
	if err != nil {
		return fmt.Errorf("setting channel enabled: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteChannel removes a channel.
func (s *Store) DeleteChannel(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM channels WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("deleting channel: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
