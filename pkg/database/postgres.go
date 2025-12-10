package database

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewConnection(dsn string) (*gorm.DB, func(), error) {
	return NewConnectionWithRetry(dsn, 10, 2*time.Second)
}

func NewConnectionWithRetry(dsn string, maxRetries int, retryInterval time.Duration) (*gorm.DB, func(), error) {
	var db *gorm.DB
	var err error

	for i := 0; i < maxRetries; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			sqlDB, sqlErr := db.DB()
			if sqlErr == nil {
				if pingErr := sqlDB.Ping(); pingErr == nil {
					sqlDB.SetMaxOpenConns(25)
					sqlDB.SetMaxIdleConns(10)
					sqlDB.SetConnMaxLifetime(5 * time.Minute)

					cleanup := func() {
						if sqlDB != nil {
							sqlDB.Close()
						}
					}
					return db, cleanup, nil
				}
			}
		}

		fmt.Printf("Failed to connect to database (attempt %d/%d): %v. Retrying in %v...\n",
			i+1, maxRetries, err, retryInterval)
		time.Sleep(retryInterval)
	}

	return nil, nil, fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
}
