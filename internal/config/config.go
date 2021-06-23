package config

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
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
	// netlog backup
	NetlogUnixSocketAddrDir  string `toml:"netlog_unix_socket_addr_dir"`
	NetlogUnixSocketFileName string `toml:"netlog_unix_socket_file_name"`
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

func Load(env, path string) (*Config, error) {
	var tomlConfig Toml
	if _, err := toml.DecodeFile(path, &tomlConfig); err != nil {
		return nil, err
	}

	cfg, err := tomlConfig.Get(env)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
