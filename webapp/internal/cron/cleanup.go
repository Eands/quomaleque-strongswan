package cron

import (
	"database/sql"
	"log"
	"time"
)

func Start(db *sql.DB, defaultRetentionDays int) {
	go func() {
		for {
			cleanupOldLogs(db, defaultRetentionDays)
			time.Sleep(24 * time.Hour)
		}
	}()
}

func cleanupOldLogs(db *sql.DB, defaultRetentionDays int) {
	var retentionDays int
	err := db.QueryRow(`SELECT COALESCE(NULLIF(value, '')::int, $1) FROM settings WHERE key = 'log_retention_days'`, defaultRetentionDays).Scan(&retentionDays)
	if err != nil {
		log.Printf("Failed to read log retention setting: %v, using default %d", err, defaultRetentionDays)
		retentionDays = defaultRetentionDays
	}

	cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)

	result, err := db.Exec(`DELETE FROM radacct WHERE acctstoptime IS NOT NULL AND acctstoptime < $1`, cutoff)
	if err != nil {
		log.Printf("Failed to cleanup old accounting logs: %v", err)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("Cleaned up %d old accounting log entries (older than %s)", rowsAffected, cutoff.Format("2006-01-02"))
	}
}
