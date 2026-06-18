package authconfig

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	configDir  = ".rootly-catalog-sync"
	configFile = "config.yaml"
)

type Config struct {
	OAuth    *OAuthData `yaml:"oauth,omitempty"`
	ClientID string     `yaml:"client_id,omitempty"`
	Scopes   []string   `yaml:"scopes,omitempty"`
}

type OAuthData struct {
	AccessToken  string    `yaml:"access_token"`
	RefreshToken string    `yaml:"refresh_token"`
	ExpiresAt    time.Time `yaml:"expires_at"`
	TokenType    string    `yaml:"token_type"`
}

func Dir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, configDir)
}

func Path() string {
	return filepath.Join(Dir(), configFile)
}

func Load() (*Config, error) {
	data, err := os.ReadFile(Path())
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	if err := os.MkdirAll(Dir(), 0700); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(Path(), data, 0600)
}
