package config

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	IsDev       bool
	Host        string
	Port        int
	Environment string
	// logging
	LogLevel    string `toml:"log_level"`
	LogsPath    string `toml:"logs_path"`
	LogToStdout bool   `toml:"log_to_stdout"`
	// aerospike
	AeroHost        string `toml:"aero_host"`
	AeroPort        int    `toml:"aero_port"`
	AeroNamespace   string `toml:"aero_namespace"`
	AeroMessagesSet string `toml:"aero_messages_set"`
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

	cfg.IsDev = false
	if env == "development" {
		cfg.IsDev = true
	}

	cfg.Environment = env

	return cfg, nil
}
