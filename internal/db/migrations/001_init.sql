CREATE TABLE IF NOT EXISTS app_metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

INSERT INTO app_metadata (key, value)
VALUES ('app_name', 'Sistema de mantenimiento')
ON CONFLICT(key) DO NOTHING;
