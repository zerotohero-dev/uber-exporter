package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	AppName    = "uber-exporter"
	ConfigFile = "config.json"
)

type IMAPConfig struct {
	Server      string `json:"server"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	PasswordCmd string `json:"password_cmd"`
}

type Config struct {
	IMAP      IMAPConfig `json:"imap"`
	Cookie    string     // loaded from cookie.txt, not JSON
	OutboxDir string     `json:"outbox_dir"`
}

func DefaultConfig() Config {
	return Config{
		IMAP: IMAPConfig{
			Server: "imap.gmail.com",
			Port:   993,
		},
		OutboxDir: "outbox",
	}
}

func ConfigDir() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("getting home directory: %w", err)
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, AppName), nil
}

func Load() (Config, error) {
	dir, err := ConfigDir()
	if err != nil {
		return Config{}, err
	}

	path := filepath.Join(dir, ConfigFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return Config{}, fmt.Errorf("reading config: %w", err)
	}

	cfg := DefaultConfig()
	if len(data) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config: %w", err)
	}

	// Load cookie from cookie.txt (plain text, avoids JSON escaping issues)
	cookiePath := filepath.Join(dir, "cookie.txt")
	if cookieData, err := os.ReadFile(cookiePath); err == nil {
		cfg.Cookie = strings.TrimSpace(string(cookieData))
	}

	return cfg, nil
}
