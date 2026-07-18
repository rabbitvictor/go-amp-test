package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config is the resolved application configuration, loaded from (in order of
// precedence): explicit env vars, a config file, then built-in defaults.
type Config struct {
	Server ServerConfig `mapstructure:"server"`
	DB     DBConfig     `mapstructure:"db"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port        string `mapstructure:"port"`
	ServiceName string `mapstructure:"service_name"`
	Version     string `mapstructure:"version"`
}

// DBConfig holds SQLite connection settings.
type DBConfig struct {
	Path         string `mapstructure:"path"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	// BusyTimeout is the SQLite busy timeout in milliseconds.
	BusyTimeout int `mapstructure:"busy_timeout"`
	// JournalMode is the SQLite journal mode (WAL, MEMORY, DELETE, ...).
	JournalMode string `mapstructure:"journal_mode"`
	// Synchronous is the SQLite synchronous pragma (NORMAL, FULL, OFF).
	Synchronous string `mapstructure:"synchronous"`
	// ForeignKeys enables SQLite foreign key enforcement.
	ForeignKeys bool `mapstructure:"foreign_keys"`
}

// Addr returns the listen address for the configured port.
func (s ServerConfig) Addr() string {
	return ":" + s.Port
}

// DSN returns the modernc.org/sqlite DSN built from the DB config.
func (c DBConfig) DSN() string {
	return fmt.Sprintf(
		"file:%s?_journal_mode=%s&_busy_timeout=%d&_synchronous=%s&_foreign_keys=%s",
		c.Path,
		strings.ToUpper(c.JournalMode),
		c.BusyTimeout,
		strings.ToUpper(c.Synchronous),
		boolStr(c.ForeignKeys),
	)
}

func boolStr(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

// Load builds the Config by merging defaults, an optional config file, and
// environment variables. The config file is optional; if absent, defaults and
// env vars are used. CONFIG_PATH, if set, points at a specific config file;
// otherwise config.{yaml,yml,json,toml} is searched in the working directory,
// ./config, and /etc/go-amp-test.
func Load() (Config, error) {
	v := viper.New()

	// --- defaults -------------------------------------------------------
	v.SetDefault("server.port", "8080")
	v.SetDefault("server.service_name", "go-amp-test")
	v.SetDefault("server.version", "0.1.0")
	v.SetDefault("db.path", "app.db")
	v.SetDefault("db.max_open_conns", 1)
	v.SetDefault("db.busy_timeout", 5000)
	v.SetDefault("db.journal_mode", "WAL")
	v.SetDefault("db.synchronous", "NORMAL")
	v.SetDefault("db.foreign_keys", true)

	// --- config file (optional) ----------------------------------------
	v.SetConfigName("config")
	for _, dir := range configSearchDirs() {
		v.AddConfigPath(dir)
	}
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		abs, err := filepath.Abs(path)
		if err != nil {
			abs = path
		}
		v.SetConfigFile(abs)
	}

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return Config{}, fmt.Errorf("read config: %w", err)
		}
		// Missing config file is fine: fall back to defaults + env.
	}

	// --- environment ----------------------------------------------------
	// Bind legacy bare env var names so existing deployments keep working.
	if err := v.BindEnv("server.port", "PORT"); err != nil {
		return Config{}, fmt.Errorf("bind env: %w", err)
	}
	if err := v.BindEnv("server.service_name", "SERVICE_NAME"); err != nil {
		return Config{}, fmt.Errorf("bind env: %w", err)
	}
	if err := v.BindEnv("server.version", "SERVICE_VERSION"); err != nil {
		return Config{}, fmt.Errorf("bind env: %w", err)
	}
	if err := v.BindEnv("db.path", "DB_PATH"); err != nil {
		return Config{}, fmt.Errorf("bind env: %w", err)
	}
	if err := v.BindEnv("db.max_open_conns", "DB_MAX_OPEN_CONNS"); err != nil {
		return Config{}, fmt.Errorf("bind env: %w", err)
	}
	if err := v.BindEnv("db.busy_timeout", "DB_BUSY_TIMEOUT"); err != nil {
		return Config{}, fmt.Errorf("bind env: %w", err)
	}
	if err := v.BindEnv("db.journal_mode", "DB_JOURNAL_MODE"); err != nil {
		return Config{}, fmt.Errorf("bind env: %w", err)
	}
	if err := v.BindEnv("db.synchronous", "DB_SYNCHRONOUS"); err != nil {
		return Config{}, fmt.Errorf("bind env: %w", err)
	}
	if err := v.BindEnv("db.foreign_keys", "DB_FOREIGN_KEYS"); err != nil {
		return Config{}, fmt.Errorf("bind env: %w", err)
	}
	// Also allow the natural SERVER_* / DB_* names when set explicitly.
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	return cfg, nil
}

func configSearchDirs() []string {
	return []string{".", "./config", "/etc/go-amp-test"}
}
