// Package config loads all DataVault configuration from environment variables.
// No configuration is hard-coded. Sensitive values (search key, DB passwords)
// are never logged.
//
// All environment variables use the DATAVAULT_ prefix per naming convention.
package config

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration for DataVault.
type Config struct {
	// Server
	HTTPPort int
	Env      string // dev | test | prod — controls safety guardrails

	// Database
	DBDriver    string // postgres | mssql | oracle
	DatabaseURL string // used for postgres (DATAVAULT_DB_DSN)

	// MSSQL connection fields
	MSSQLHost string
	MSSQLPort int
	MSSQLUser string
	MSSQLPass string
	MSSQLDB   string

	// Oracle connection fields
	OracleDSN  string
	OracleUser string
	OraclePass string

	// HSM
	HSMMode string // stub | pkcs11

	// HMAC search key — never log
	HMACKey []byte

	// DEK cache
	DEKCacheTTL time.Duration

	// Database connection pool — applied to all DB drivers.
	// See DATAVAULT_DB_* env vars.
	DBMaxConns          int           // max open connections (default 20)
	DBMinConns          int           // min idle connections / pgx MinConns (default 2)
	DBConnMaxLifetime   time.Duration // rotate connections after this age (default 30m)
	DBConnMaxIdleTime   time.Duration // release idle connections after this age (default 5m)
	DBHealthCheckPeriod time.Duration // pgx only: ping idle conns at this interval (default 1m)

	// Logging
	LogLevel string

	// API authentication — never log
	APIKey string
}

// Load reads configuration from environment variables and returns a validated
// Config. Exits the process if required values are missing or invalid.
func Load() *Config {
	searchKeyHex := requireEnv("DATAVAULT_SEARCH_KEY")
	searchKey, err := hex.DecodeString(searchKeyHex)
	if err != nil || len(searchKey) != 32 {
		fatalf("DATAVAULT_SEARCH_KEY must be a 64-character hex string (32 bytes), got length %d", len(searchKeyHex))
	}

	ttlStr := getEnvOrDefault("DATAVAULT_DEK_CACHE_TTL", "5m")
	ttl, err := time.ParseDuration(ttlStr)
	if err != nil {
		fatalf("invalid DATAVAULT_DEK_CACHE_TTL: %v", err)
	}

	httpPort, _ := strconv.Atoi(getEnvOrDefault("DATAVAULT_APP_PORT", "8080"))
	mssqlPort, _ := strconv.Atoi(getEnvOrDefault("DATAVAULT_MSSQL_PORT", "1433"))

	dbMaxConns, _ := strconv.Atoi(getEnvOrDefault("DATAVAULT_DB_MAX_CONNS", "20"))
	dbMinConns, _ := strconv.Atoi(getEnvOrDefault("DATAVAULT_DB_MIN_CONNS", "2"))
	dbConnMaxLifetime := parseDurationOrDefault("DATAVAULT_DB_CONN_MAX_LIFETIME", 30*time.Minute)
	dbConnMaxIdleTime := parseDurationOrDefault("DATAVAULT_DB_CONN_MAX_IDLE_TIME", 5*time.Minute)
	dbHealthCheckPeriod := parseDurationOrDefault("DATAVAULT_DB_HEALTH_CHECK_PERIOD", 1*time.Minute)

	return &Config{
		HTTPPort:            httpPort,
		Env:                 getEnvOrDefault("DATAVAULT_ENV", "dev"),
		DBDriver:            getEnvOrDefault("DATAVAULT_DB_DRIVER", "postgres"),
		DatabaseURL:         os.Getenv("DATAVAULT_DB_DSN"),
		MSSQLHost:           os.Getenv("DATAVAULT_MSSQL_HOST"),
		MSSQLPort:           mssqlPort,
		MSSQLUser:           os.Getenv("DATAVAULT_MSSQL_USER"),
		MSSQLPass:           os.Getenv("DATAVAULT_MSSQL_PASS"),
		MSSQLDB:             os.Getenv("DATAVAULT_MSSQL_DB"),
		OracleDSN:           os.Getenv("DATAVAULT_ORACLE_DSN"),
		OracleUser:          os.Getenv("DATAVAULT_ORACLE_USER"),
		OraclePass:          os.Getenv("DATAVAULT_ORACLE_PASS"),
		HSMMode:             getEnvOrDefault("DATAVAULT_HSM_MODE", "stub"),
		HMACKey:             searchKey,
		DEKCacheTTL:         ttl,
		DBMaxConns:          dbMaxConns,
		DBMinConns:          dbMinConns,
		DBConnMaxLifetime:   dbConnMaxLifetime,
		DBConnMaxIdleTime:   dbConnMaxIdleTime,
		DBHealthCheckPeriod: dbHealthCheckPeriod,
		LogLevel:            getEnvOrDefault("DATAVAULT_LOG_LEVEL", "info"),
		APIKey:              requireEnv("DATAVAULT_API_KEY"),
	}
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fatalf("required environment variable %s is not set", key)
	}
	return v
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "config: "+format+"\n", args...)
	os.Exit(1)
}

func parseDurationOrDefault(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		fatalf("invalid %s: %v", key, err)
	}
	return d
}
