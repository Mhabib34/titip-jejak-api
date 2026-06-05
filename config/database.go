package config

import (
	"titip-jejak-api/internal/logger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func NewDB(cfg *Config) *gorm.DB {
	log := logger.Get()

	logLevel := gormlogger.Silent
	if cfg.Env == "development" {
		logLevel = gormlogger.Info
	}

	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		Logger: gormlogger.Default.LogMode(logLevel),
	})
	if err != nil {
		logger.Fatal("failed to connect to database", "error", err)
	}

	log.Info("database connected", "env", cfg.Env)
	return db
}