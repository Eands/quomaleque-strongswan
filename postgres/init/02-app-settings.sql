CREATE TABLE IF NOT EXISTS settings (
    key VARCHAR(64) PRIMARY KEY,
    value TEXT NOT NULL DEFAULT ''
);

INSERT INTO settings (key, value) VALUES ('log_retention_days', '90')
ON CONFLICT (key) DO NOTHING;
