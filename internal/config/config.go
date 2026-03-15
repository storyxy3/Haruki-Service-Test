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
	Server        ServerConfig        `yaml:"server"`
	DrawingAPI    DrawingAPIConfig    `yaml:"drawing_api"`
	MasterData    MasterDataConfig    `yaml:"masterdata"`
	Assets        AssetsConfig        `yaml:"assets"`
	UserData      UserDataConfig      `yaml:"user_data"`
	Log           LogConfig           `yaml:"log"`
	HarukiCloud   HarukiCloudConfig   `yaml:"haruki_cloud"`
	DeckRecommend DeckRecommendConfig `yaml:"deck_recommend"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type DrawingAPIConfig struct {
	BaseURL    string        `yaml:"base_url"`
	Timeout    time.Duration `yaml:"timeout"`
	RetryCount int           `yaml:"retry_count"`
	OutputDir  string        `yaml:"output_dir"`
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
	Path          string `yaml:"path"`
	MusicMetaPath string `yaml:"music_meta_path"`
	MysekaiPath   string `yaml:"mysekai_path"`
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type HarukiCloudConfig struct {
	SekaiDB          DatabaseConfig `yaml:"sekai_db"`
	UseLocalEventSrc bool           `yaml:"use_local_event_source"`
	UseLocalCardSrc  bool           `yaml:"use_local_card_source"`
	Region           string         `yaml:"region"`
	SecondaryRegion  string         `yaml:"secondary_region"`
}

type DatabaseConfig struct {
	Driver   string `yaml:"driver"`
	DSN      string `yaml:"dsn"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	SSLMode  string `yaml:"sslmode"`
}

type DeckRecommendConfig struct {
	Enabled        bool                        `yaml:"enabled"`
	UseLocalEngine bool                        `yaml:"use_local_engine"`
	LocalPoolSize  int                         `yaml:"local_pool_size"`
	Timeout        time.Duration               `yaml:"timeout"`
	Servers        []DeckRecommendServerConfig `yaml:"servers"`
	DefaultAlgs    []string                    `yaml:"default_algs"`
}

type DeckRecommendServerConfig struct {
	URL    string `yaml:"url"`
	Weight int    `yaml:"weight"`
}

// Load loads the configuration from the given path.
func Load(path string) (*Config, error) {
	f, err := openConfig(path)
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

func openConfig(path string) (*os.File, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("empty config path")
	}

	if filepath.IsAbs(path) {
		return os.Open(path)
	}

	if f, err := os.Open(path); err == nil {
		return f, nil
	}

	exePath, err := os.Executable()
	if err != nil {
		return nil, os.ErrNotExist
	}

	exeDir := filepath.Dir(exePath)
	return os.Open(filepath.Join(exeDir, path))
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
