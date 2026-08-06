package config

import (
	"os"
	"strconv"
)

type Config struct {
	DBHost           string
	DBPort           string
	DBUser           string
	DBPassword       string
	DBName           string
	ViciAddr         string
	ViciTimeout      int
	ViciReconnect    int
	HTTPSPort        string
	SSLCertPath      string
	SSLKeyPath       string
	LogRetentionDays int
	AppUser          string
	AppPassword      string
	RadiusSecret     string
}

func Load() *Config {
	return &Config{
		DBHost:           getEnv("DB_HOST", "postgres"),
		DBPort:           getEnv("DB_PORT", "5432"),
		DBUser:           getEnv("DB_USER", "radius"),
		DBPassword:       getEnv("DB_PASSWORD", "change_me_radius_password"),
		DBName:           getEnv("DB_NAME", "radius"),
		ViciAddr:         getEnv("STRONGSWAN_VICI_ADDR", "strongswan:4502"),
		ViciTimeout:      getEnvInt("VICI_CONNECT_TIMEOUT", 10),
		ViciReconnect:    getEnvInt("VICI_RECONNECT_INTERVAL", 5),
		HTTPSPort:        getEnv("HTTPS_PORT", "443"),
		SSLCertPath:      getEnv("SSL_CERT_PATH", ""),
		SSLKeyPath:       getEnv("SSL_KEY_PATH", ""),
		LogRetentionDays: getEnvInt("LOG_RETENTION_DAYS", 90),
		AppUser:          getEnv("APP_USER", "admin"),
		AppPassword:      getEnv("APP_PASSWORD", "change_me_admin_password"),
		RadiusSecret:     getEnv("RADIUS_SECRET", "change_me_radius_secret"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}
