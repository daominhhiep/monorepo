// Package db is a thin wrapper around gorm.Open with sane defaults
// (logger silenced, connection pool tuned, prepared-stmt caching off).
package db

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type Config struct {
	URL             string        `kong:"name='postgres-url',env='POSTGRES_URL',required"`
	MaxOpenConns    int           `kong:"name='postgres-max-open',env='POSTGRES_MAX_OPEN',default='25'"`
	MaxIdleConns    int           `kong:"name='postgres-max-idle',env='POSTGRES_MAX_IDLE',default='5'"`
	ConnMaxLifetime time.Duration `kong:"name='postgres-conn-lifetime',env='POSTGRES_CONN_LIFETIME',default='30m'"`
	AutoMigrate     bool          `kong:"name='auto-migrate',env='AUTO_MIGRATE',help='Run GORM AutoMigrate on boot (dev only)'"`
}

func Open(cfg Config) (*gorm.DB, error) {
	if cfg.URL == "" {
		return nil, errors.New("postgres url required")
	}
	g, err := gorm.Open(postgres.Open(cfg.URL), &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Warn),
		PrepareStmt:                              false,
		DisableForeignKeyConstraintWhenMigrating: false,
	})
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	sqlDB, err := g.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	return g, nil
}

// Ping is the readiness probe primitive.
func Ping(g *gorm.DB) error {
	sqlDB, err := g.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
