-- migrations/001_initial.sql
-- Пользователи
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,  -- bcrypt
    is_active BOOLEAN DEFAULT 1
);

-- Сессии VPN
CREATE TABLE sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL,
    framed_ip TEXT,
    session_id TEXT UNIQUE,
    start_time DATETIME DEFAULT CURRENT_TIMESTAMP,
    end_time DATETIME,
    input_octets INTEGER DEFAULT 0,
    output_octets INTEGER DEFAULT 0,
    FOREIGN KEY (username) REFERENCES users(username)
);

-- Логи RADIUS
CREATE TABLE radius_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT,
    request_type TEXT,  -- Access-Request, Accounting-Request
    result TEXT,        -- Accept, Reject
    client_ip TEXT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Настройки системы
CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Индексы
CREATE INDEX idx_sessions_username ON sessions(username);
CREATE INDEX idx_sessions_session_id ON sessions(session_id);
CREATE INDEX idx_radius_logs_username ON radius_logs(username);
CREATE INDEX idx_radius_logs_timestamp ON radius_logs(timestamp);