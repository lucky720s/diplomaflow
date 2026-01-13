package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Options struct {
	MaxRetries    int
	RetryInterval time.Duration
	PingTimeout   time.Duration

	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration

	Logger     *log.Logger
	GormConfig *gorm.Config
}

func DefaultOptions() Options {
	return Options{
		MaxRetries:      10,
		RetryInterval:   2 * time.Second,
		PingTimeout:     2 * time.Second,
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
		Logger:          nil,
		GormConfig:      &gorm.Config{},
	}
}

func NewConnection(dsn string) (*gorm.DB, func(), error) {
	return NewConnectionWithOptions(dsn, DefaultOptions())
}

func NewConnectionWithRetry(dsn string, maxRetries int, retryInterval time.Duration) (*gorm.DB, func(), error) {
	opts := DefaultOptions()
	opts.MaxRetries = maxRetries
	opts.RetryInterval = retryInterval
	return NewConnectionWithOptions(dsn, opts)
}

func NewConnectionWithOptions(dsn string, opts Options) (*gorm.DB, func(), error) {
	if opts.MaxRetries <= 0 {
		opts.MaxRetries = 1
	}
	if opts.RetryInterval <= 0 {
		opts.RetryInterval = 500 * time.Millisecond
	}
	if opts.PingTimeout <= 0 {
		opts.PingTimeout = 2 * time.Second
	}
	if opts.GormConfig == nil {
		opts.GormConfig = &gorm.Config{}
	}

	l := opts.Logger
	if l == nil {
		l = log.Default()
	}

	var lastErr error

	for attempt := 1; attempt <= opts.MaxRetries; attempt++ {
		db, err := gorm.Open(postgres.Open(dsn), opts.GormConfig)
		if err != nil {
			lastErr = fmt.Errorf("gorm open: %w", err)
			l.Printf("DB connect failed (attempt %d/%d): %v", attempt, opts.MaxRetries, lastErr)
			time.Sleep(opts.RetryInterval)
			continue
		}

		sqlDB, err := db.DB()
		if err != nil {
			lastErr = fmt.Errorf("db.DB(): %w", err)
			l.Printf("DB connect failed (attempt %d/%d): %v", attempt, opts.MaxRetries, lastErr)
			_ = sqlDB.Close()
			time.Sleep(opts.RetryInterval)
			continue
		}

		pingCtx, cancel := context.WithTimeout(context.Background(), opts.PingTimeout)
		pingErr := sqlDB.PingContext(pingCtx)
		cancel()
		if pingErr != nil {
			lastErr = fmt.Errorf("ping: %w", pingErr)
			l.Printf("DB ping failed (attempt %d/%d): %v", attempt, opts.MaxRetries, lastErr)
			_ = sqlDB.Close()
			time.Sleep(opts.RetryInterval)
			continue
		}

		// Pool tuning
		if opts.MaxOpenConns > 0 {
			sqlDB.SetMaxOpenConns(opts.MaxOpenConns)
		}
		if opts.MaxIdleConns > 0 {
			sqlDB.SetMaxIdleConns(opts.MaxIdleConns)
		}
		if opts.ConnMaxLifetime > 0 {
			sqlDB.SetConnMaxLifetime(opts.ConnMaxLifetime)
		}
		if opts.ConnMaxIdleTime > 0 {
			sqlDB.SetConnMaxIdleTime(opts.ConnMaxIdleTime)
		}

		cleanup := func() { _ = sqlDB.Close() }
		return db, cleanup, nil
	}

	return nil, nil, fmt.Errorf("failed to connect to database after %d attempts: %w", opts.MaxRetries, lastErr)
}
