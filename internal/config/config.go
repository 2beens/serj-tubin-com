package config

import (
	"fmt"
	"strings"
)

type Config struct {
	Port int
	// logging
	LogLevel    string `toml:"log_level"`
	LogsPath    string `toml:"logs_path"`
	LogToStdout bool   `toml:"log_to_stdout"`
	// aerospike
	AeroHost        string `toml:"aero_host"`
	AeroPort        int    `toml:"aero_port"`
	AeroNamespace   string `toml:"aero_namespace"`
	AeroMessagesSet string `toml:"aero_messages_set"`
}

type Toml struct {
	Development *Config
	Production  *Config
}

func (t *Toml) Get(env string) (*Config, error) {
	switch strings.ToLower(env) {
	case "dev", "development":
		return t.Development, nil
	case "prod", "production":
		return t.Production, nil
	default:
		return nil, fmt.Errorf("unknown env: %s", env)
	}
}
