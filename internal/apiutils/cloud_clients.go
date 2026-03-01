package apiutils

import (
	"fmt"
	"log/slog"

	"Haruki-Service-API/internal/config"

	sekai "haruki-cloud/database/sekai"
)

// CloudClients 封装 Haruki-Cloud 的数据库客户端，集中由 API utils 统一管理。
type CloudClients struct {
	Sekai *sekai.Client
}

// InitCloudClients 根据配置初始化 Haruki-Cloud 的数据库客户端（目前仅支持 Sekai）。
func InitCloudClients(cfg config.HarukiCloudConfig, logger *slog.Logger) (*CloudClients, error) {
	clients := &CloudClients{}

	if cfg.SekaiDB.Driver == "" {
		if logger != nil {
			logger.Info("Haruki Cloud Sekai DB driver not configured, skipping connection")
		}
		return clients, nil
	}

	dsn := cfg.SekaiDB.DSN
	if dsn == "" {
		var err error
		dsn, err = buildDSN(cfg.SekaiDB)
		if err != nil {
			return nil, err
		}
	}

	sekaiClient, err := sekai.Open(cfg.SekaiDB.Driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect Sekai DB: %w", err)
	}
	if logger != nil {
		logger.Info("Connected to Haruki Cloud Sekai DB")
	}
	clients.Sekai = sekaiClient
	return clients, nil
}

// Close 关闭所有数据库连接。
func (c *CloudClients) Close() {
	if c == nil {
		return
	}
	if c.Sekai != nil {
		_ = c.Sekai.Close()
		c.Sekai = nil
	}
}

func buildDSN(db config.DatabaseConfig) (string, error) {
	switch db.Driver {
	case "postgres", "postgresql":
		if db.Host == "" || db.Database == "" || db.User == "" {
			return "", fmt.Errorf("postgres config requires host, database, user")
		}
		port := db.Port
		if port == 0 {
			port = 5432
		}
		sslMode := db.SSLMode
		if sslMode == "" {
			sslMode = "disable"
		}
		password := db.Password
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			db.Host, port, db.User, password, db.Database, sslMode), nil
	case "mysql":
		if db.Host == "" || db.Database == "" || db.User == "" {
			return "", fmt.Errorf("mysql config requires host, database, user")
		}
		port := db.Port
		if port == 0 {
			port = 3306
		}
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
			db.User, db.Password, db.Host, port, db.Database), nil
	case "sqlite", "sqlite3":
		if db.Database == "" {
			return "", fmt.Errorf("sqlite config requires database path")
		}
		return db.Database, nil
	default:
		return "", fmt.Errorf("unsupported driver for DSN build: %s", db.Driver)
	}
}
