package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type SettingsHandler struct {
	DB *sql.DB
}

func NewSettingsHandler(db *sql.DB) *SettingsHandler {
	return &SettingsHandler{DB: db}
}

func (h *SettingsHandler) SettingsPage(c *gin.Context) {
	var logRetentionDays string
	err := h.DB.QueryRow(`SELECT value FROM settings WHERE key = 'log_retention_days'`).Scan(&logRetentionDays)
	if err != nil {
		logRetentionDays = "90"
	}

	c.HTML(http.StatusOK, "settings.html", H("settings", gin.H{"log_retention_days": logRetentionDays}))
}

func (h *SettingsHandler) UpdateSettings(c *gin.Context) {
	logRetentionDays := c.PostForm("log_retention_days")

	days, err := strconv.Atoi(logRetentionDays)
	if err != nil || days < 1 {
		c.HTML(http.StatusBadRequest, "settings.html", H("settings", gin.H{
			"log_retention_days": logRetentionDays,
			"error":              "Log retention days must be a positive number",
		}))
		return
	}

	_, err = h.DB.Exec(`INSERT INTO settings (key, value) VALUES ('log_retention_days', $1)
		ON CONFLICT (key) DO UPDATE SET value = $1`, logRetentionDays)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "settings.html", H("settings", gin.H{
			"log_retention_days": logRetentionDays,
			"error":              "Failed to update settings: " + err.Error(),
		}))
		return
	}

	c.HTML(http.StatusOK, "settings.html", H("settings", gin.H{
		"log_retention_days": logRetentionDays,
		"success":            "Settings updated successfully",
	}))
}
