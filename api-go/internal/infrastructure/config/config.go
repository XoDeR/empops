// Package config loads the YAML application configuration (config/app.*.yaml)
// plus the enabled-modules list (config/modules.yaml), with environment
// variable overrides for secrets.
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the root application configuration.
type Config struct {
	Env  string     `yaml:"env"`
	HTTP HTTPConfig `yaml:"http"`
	JWT  JWTConfig  `yaml:"jwt"`
	CORS CORSConfig `yaml:"cors"`
	Log  LogConfig  `yaml:"log"`
	DB   DBConfig   `yaml:"db"`
}

// HTTPConfig configures the API HTTP server.
type HTTPConfig struct {
	Port int `yaml:"port"`
}

// JWTConfig configures token issuance/verification.
type JWTConfig struct {
	Secret          string `yaml:"secret"`
	Issuer          string `yaml:"issuer"`
	Audience        string `yaml:"audience"`
	AccessTokenTTL  string `yaml:"access_token_ttl"`
	RefreshTokenTTL string `yaml:"refresh_token_ttl"`
}

// AccessTTL parses AccessTokenTTL, defaulting to 15m.
func (j JWTConfig) AccessTTL() time.Duration {
	return parseDurationDefault(j.AccessTokenTTL, 15*time.Minute)
}

// RefreshTTL parses RefreshTokenTTL, defaulting to 7d.
func (j JWTConfig) RefreshTTL() time.Duration {
	return parseDurationDefault(j.RefreshTokenTTL, 7*24*time.Hour)
}

func parseDurationDefault(raw string, fallback time.Duration) time.Duration {
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return d
}

// CORSConfig lists the origins allowed to call the API from a browser.
type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
}

// LogConfig configures the structured logger.
type LogConfig struct {
	Level string `yaml:"level"`
}

// DBConfig configures the optional PostgreSQL connection (unused by the
// Step 0 stub auth flow).
type DBConfig struct {
	DSN string `yaml:"dsn"`
}

// ModulesConfig lists which vertical modules are enabled.
type ModulesConfig struct {
	Enabled []string `yaml:"enabled"`
}

// Load reads and parses the app config file at path, applying environment
// variable overrides for secrets that should not live in YAML long-term.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}

	applyEnvOverrides(&cfg)
	applyDefaults(&cfg)

	return &cfg, nil
}

// LoadModules reads and parses the modules config file at path.
func LoadModules(path string) (*ModulesConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	var cfg ModulesConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}
	return &cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("EMPOPS_JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
	}
	if v := os.Getenv("EMPOPS_DB_DSN"); v != "" {
		cfg.DB.DSN = v
	}
	if v := os.Getenv("EMPOPS_HTTP_PORT"); v != "" {
		if port, err := parsePort(v); err == nil {
			cfg.HTTP.Port = port
		}
	}
}

func parsePort(raw string) (int, error) {
	var port int
	_, err := fmt.Sscanf(raw, "%d", &port)
	return port, err
}

func applyDefaults(cfg *Config) {
	if cfg.HTTP.Port == 0 {
		cfg.HTTP.Port = 8080
	}
	if cfg.JWT.Issuer == "" {
		cfg.JWT.Issuer = "empops"
	}
	if cfg.JWT.Audience == "" {
		cfg.JWT.Audience = "empops-web"
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	if len(cfg.CORS.AllowedOrigins) == 0 {
		cfg.CORS.AllowedOrigins = []string{"http://localhost:5173", "http://localhost:3000"}
	}
}
