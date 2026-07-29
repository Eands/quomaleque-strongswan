package db

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	ID           int
	Username     string
	PasswordHash string
	IsActive     bool
}

type Session struct {
	ID           int
	Username     string
	FramedIP     string
	SessionID    string
	StartTime    time.Time
	EndTime      *time.Time
	InputOctets  int64
	OutputOctets int64
}

type RadiusLog struct {
	ID          int
	Username    string
	RequestType string
	Result      string
	ClientIP    string
	Timestamp   time.Time
}

type Setting struct {
	Key       string
	Value     string
	UpdatedAt time.Time
}

type Database struct {
	db *sql.DB
}

func NewDatabase(path string) (*Database, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	db.Exec("PRAGMA foreign_keys = ON")

	return &Database{db: db}, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) GetUserByUsername(username string) (*User, error) {
	var user User
	query := `SELECT id, username, password_hash, is_active
              FROM users WHERE username = ?`
	err := d.db.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.IsActive,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *Database) GetUserByID(id int) (*User, error) {
	var user User
	query := `SELECT id, username, password_hash, is_active
              FROM users WHERE id = ?`
	err := d.db.QueryRow(query, id).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.IsActive,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *Database) CreateUser(username, passwordHash string) error {
	query := `INSERT INTO users (username, password_hash)
              VALUES (?, ?)`
	_, err := d.db.Exec(query, username, passwordHash)
	return err
}

func (d *Database) UpdateUser(id int, username, passwordHash string) error {
	query := `UPDATE users SET username = ?, password_hash = ? WHERE id = ?`
	_, err := d.db.Exec(query, username, passwordHash, id)
	return err
}

func (d *Database) DeleteUser(id int) error {
	query := `DELETE FROM users WHERE id = ?`
	_, err := d.db.Exec(query, id)
	return err
}

func (d *Database) ToggleUserActive(id int) error {
	query := `UPDATE users SET is_active = NOT is_active WHERE id = ?`
	_, err := d.db.Exec(query, id)
	return err
}

func (d *Database) ListUsers() ([]User, error) {
	rows, err := d.db.Query(
		`SELECT id, username, password_hash, is_active
		FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.IsActive)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (d *Database) CountUsers() (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

func (d *Database) InsertSessionLog(username, framedIP, sessionID string) error {
	query := `INSERT INTO sessions (username, framed_ip, session_id, start_time)
              VALUES (?, ?, ?, CURRENT_TIMESTAMP)`
	_, err := d.db.Exec(query, username, framedIP, sessionID)
	return err
}

func (d *Database) UpdateSessionStop(sessionID string, inputOctets, outputOctets int64) error {
	query := `UPDATE sessions SET end_time = CURRENT_TIMESTAMP,
              input_octets = ?, output_octets = ? WHERE session_id = ?`
	_, err := d.db.Exec(query, inputOctets, outputOctets, sessionID)
	return err
}

func (d *Database) UpdateSessionInterim(sessionID string, inputOctets, outputOctets int64) error {
	query := `UPDATE sessions SET input_octets = ?, output_octets = ?
              WHERE session_id = ?`
	_, err := d.db.Exec(query, inputOctets, outputOctets, sessionID)
	return err
}

func (d *Database) ListSessions() ([]Session, error) {
	rows, err := d.db.Query(
		`SELECT id, username, framed_ip, session_id, start_time, end_time,
		 input_octets, output_octets
		 FROM sessions ORDER BY start_time DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		err := rows.Scan(&s.ID, &s.Username, &s.FramedIP, &s.SessionID,
			&s.StartTime, &s.EndTime, &s.InputOctets, &s.OutputOctets)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (d *Database) GetActiveSessionCount() (int, error) {
	var count int
	err := d.db.QueryRow(
		"SELECT COUNT(*) FROM sessions WHERE end_time IS NULL").Scan(&count)
	return count, err
}

func (d *Database) InsertRadiusLog(username, requestType, result, clientIP string) error {
	query := `INSERT INTO radius_logs (username, request_type, result, client_ip, timestamp)
              VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)`
	_, err := d.db.Exec(query, username, requestType, result, clientIP)
	return err
}

func (d *Database) ListLogs() ([]RadiusLog, error) {
	rows, err := d.db.Query(
		`SELECT id, username, request_type, result, client_ip, timestamp
		 FROM radius_logs ORDER BY timestamp DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []RadiusLog
	for rows.Next() {
		var l RadiusLog
		err := rows.Scan(&l.ID, &l.Username, &l.RequestType, &l.Result,
			&l.ClientIP, &l.Timestamp)
		if err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

func (d *Database) GetSetting(key string) (string, error) {
	var value string
	err := d.db.QueryRow(
		"SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (d *Database) SetSetting(key, value string) error {
	query := `INSERT OR REPLACE INTO settings (key, value, updated_at)
              VALUES (?, ?, CURRENT_TIMESTAMP)`
	_, err := d.db.Exec(query, key, value)
	return err
}

func (d *Database) GetAllSettings() ([]Setting, error) {
	rows, err := d.db.Query(
		`SELECT key, value, updated_at FROM settings ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []Setting
	for rows.Next() {
		var s Setting
		err := rows.Scan(&s.Key, &s.Value, &s.UpdatedAt)
		if err != nil {
			return nil, err
		}
		settings = append(settings, s)
	}
	return settings, nil
}
