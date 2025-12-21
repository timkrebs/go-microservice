package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"github.com/timkrebs/image-processor/internal/metrics"
)

// DB wraps the sql.DB connection
type DB struct {
	*sql.DB
	metrics *metrics.DatabaseMetrics
}

// New creates a new database connection
func New(databaseURL string, maxConns int) (*DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(maxConns)
	db.SetMaxIdleConns(maxConns / 2)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{DB: db}, nil
}

// SetMetrics injects metrics collectors into database client
func (db *DB) SetMetrics(m *metrics.DatabaseMetrics) {
	db.metrics = m

	// Update connection pool metrics periodically
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			if db.metrics != nil {
				stats := db.Stats()
				db.metrics.ConnectionsActive.Set(float64(stats.OpenConnections))
			}
		}
	}()
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}

// Health checks if the database is healthy
func (db *DB) Health(ctx context.Context) error {
	return db.PingContext(ctx)
}
