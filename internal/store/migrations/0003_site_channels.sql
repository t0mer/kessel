CREATE TABLE site_channels (
    site_id    INTEGER NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    channel_id INTEGER NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    PRIMARY KEY (site_id, channel_id)
);
CREATE INDEX idx_site_channels_channel ON site_channels(channel_id);

-- Preserve prior behaviour: existing channels notified every site, so link them all.
INSERT INTO site_channels (site_id, channel_id)
SELECT s.id, c.id FROM sites s CROSS JOIN channels c;
