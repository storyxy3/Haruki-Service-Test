package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	DrawingAPI DrawingAPIConfig `yaml:"drawing_api"`
	MasterData MasterDataConfig `yaml:"masterdata"`
	Assets     AssetsConfig     `yaml:"assets"`
	UserData   UserDataConfig   `yaml:"user_data"`
	Log        LogConfig        `yaml:"log"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type DrawingAPIConfig struct {
	BaseURL    string        `yaml:"base_url"`
	Timeout    time.Duration `yaml:"timeout"`
	RetryCount int           `yaml:"retry_count"`
}

type MasterDataConfig struct {
	Dir          string `yaml:"dir"`
	CacheEnabled bool   `yaml:"cache_enabled"`
}

type AssetsConfig struct {
	Dir        string   `yaml:"dir"`
	LegacyDirs []string `yaml:"legacy_dirs"`
}

type UserDataConfig struct {
	Path string `yaml:"path"`
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// Load loads the configuration from the given path.
func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}
	return &cfg, nil
}

// Roots returns combined primary+legacy asset directories (cleaned).
func (a AssetsConfig) Roots() []string {
	var roots []string
	appendRoot := func(path string) {
		clean := strings.TrimSpace(path)
		if clean == "" {
			return
		}
		clean = filepath.ToSlash(filepath.Clean(clean))
		if clean == "." {
			return
		}
		roots = append(roots, clean)
	}
	appendRoot(a.Dir)
	for _, legacy := range a.LegacyDirs {
		appendRoot(legacy)
	}
	return roots
}
