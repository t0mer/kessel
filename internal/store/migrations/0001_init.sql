CREATE TABLE sites (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL,
    slug       TEXT    NOT NULL UNIQUE,
    url        TEXT    NOT NULL,
    strategy   TEXT    NOT NULL CHECK (strategy IN ('mobile','desktop','both')),
    enabled    INTEGER NOT NULL DEFAULT 1,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE schedules (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id    INTEGER NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    cron_expr  TEXT    NOT NULL,
    enabled    INTEGER NOT NULL DEFAULT 1,
    created_at INTEGER NOT NULL
);
CREATE INDEX idx_schedules_site ON schedules(site_id);

CREATE TABLE runs (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id        INTEGER NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    strategy       TEXT    NOT NULL,
    started_at     INTEGER NOT NULL,
    finished_at    INTEGER,
    status         TEXT    NOT NULL,
    perf           REAL,
    accessibility  REAL,
    best_practices REAL,
    seo            REAL,
    lcp_ms         REAL,
    cls            REAL,
    tbt_ms         REAL,
    fcp_ms         REAL,
    si_ms          REAL,
    tti_ms         REAL,
    raw_json_gz    BLOB,
    report_path    TEXT,
    error          TEXT
);
CREATE INDEX idx_runs_site_strategy_started ON runs(site_id, strategy, started_at);

CREATE TABLE threshold_rules (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id  INTEGER NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    category TEXT    NOT NULL,
    mode     TEXT    NOT NULL CHECK (mode IN ('absolute','delta')),
    value    REAL    NOT NULL,
    enabled  INTEGER NOT NULL DEFAULT 1
);
CREATE INDEX idx_threshold_rules_site ON threshold_rules(site_id);

CREATE TABLE channels (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    type             TEXT    NOT NULL,
    name             TEXT    NOT NULL,
    config_encrypted BLOB,
    enabled          INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE notifications_log (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id     INTEGER REFERENCES runs(id) ON DELETE CASCADE,
    channel_id INTEGER REFERENCES channels(id) ON DELETE SET NULL,
    rule_id    INTEGER REFERENCES threshold_rules(id) ON DELETE SET NULL,
    sent_at    INTEGER NOT NULL,
    status     TEXT    NOT NULL,
    error      TEXT
);
CREATE INDEX idx_notifications_log_run ON notifications_log(run_id);

CREATE TABLE settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
