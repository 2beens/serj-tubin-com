package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Host        string
	Port        int
	Environment string
	// logging
	LogLevel    string `toml:"log_level"`
	LogsPath    string `toml:"logs_path"`
	LogToStdout bool   `toml:"log_to_stdout"`
	// netlog backup
	NetlogUnixSocketAddrDir  string `toml:"netlog_unix_socket_addr_dir"`
	NetlogUnixSocketFileName string `toml:"netlog_unix_socket_file_name"`
	// prometheus metrics
	PrometheusMetricsPort string `toml:"prometheus_metrics_port"`
	PrometheusMetricsHost string `toml:"prometheus_metrics_host"`
	// postgres db
	PostgresPort   string `toml:"postgres_port"`
	PostgresHost   string `toml:"postgres_host"`
	PostgresDBName string `toml:"postgres_db_name"`
	// Redis
	RedisHost string `toml:"redis_host"`
	RedisPort string `toml:"redis_port"`
	// Quotes
	QuotesCsvPath string `toml:"quotes_csv_path"`
	// Sentry
	SentryEnabled bool   `toml:"sentry_enabled"`
	SentryDSN     string // loaded from env. var.
	// GymStats
	GymStatsDiskApiRootPath string `toml:"gymstats_disk_api_root_path"`
	// Spotify Tracker
	SpotifyRedirectURI                string `toml:"spotify_redirect_uri"`
	PostAuthRedirectURL               string `toml:"post_auth_redirect_url"`
	SpotifyTrackerFireIntervalMinutes int    `toml:"spotify_tracker_fire_interval_minutes"`
	// Other
	LoginRateLimitAllowedPerMin int `toml:"login_rate_limit_allowed_per_min"`
	// MCP (Model Context Protocol): optional secret for /mcp endpoint. If set, only requests
	// with Authorization: Bearer <MCPSecret> (or X-MCP-Secret header) can access /mcp.
	// Can also be set via MCP_SECRET env. If empty, /mcp uses same auth as API (X-SERJ-TOKEN + login).
	MCPSecret string `toml:"mcp_secret"`
}

type Toml struct {
	DockerDev   *Config
	Development *Config
	Production  *Config
}

func (t *Toml) Get(env string) (*Config, error) {
	switch strings.ToLower(env) {
	case "dev", "development":
		return t.Development, nil
	case "prod", "production":
		return t.Production, nil
	case "ddev", "dockerdev":
		return t.DockerDev, nil
	default:
		return nil, fmt.Errorf("unknown env: %s", env)
	}
}

func Load(env, path string) (*Config, error) {
	switch env {
	case "prod":
		env = "production"
	case "dev":
		env = "development"
	case "ddev":
		env = "dockerdev"
	}

	var tomlConfig Toml
	if _, err := toml.DecodeFile(path, &tomlConfig); err != nil {
		return nil, err
	}

	cfg, err := tomlConfig.Get(env)
	if err != nil {
		return nil, err
	}

	cfg.Environment = env

	// Optional env override for MCP secret (so it is not stored in config.toml).
	if v := os.Getenv("MCP_SECRET"); v != "" {
		cfg.MCPSecret = v
	}

	return cfg, nil
}
