package platform

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/alicebob/miniredis/v2"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// EmbeddedDatabases holds references to the in-process Postgres and Redis
// instances so they can be stopped on shutdown.
type EmbeddedDatabases struct {
	pg       *embeddedpostgres.EmbeddedPostgres
	minired  *miniredis.Miniredis
	Pool     *pgxpool.Pool
	Redis    *redis.Client
}

// StartEmbeddedDatabases spins up an embedded Postgres and an in-memory Redis.
// The returned EmbeddedDatabases must be stopped via Stop() on shutdown.
func StartEmbeddedDatabases(ctx context.Context) (*EmbeddedDatabases, error) {
	// ---- Embedded PostgreSQL ------------------------------------------------
	port, err := freePort()
	if err != nil {
		return nil, fmt.Errorf("find free port: %w", err)
	}

	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Port(uint32(port)).
			Database("eol").
			Username("eol").
			Password("eoldev").
			Version(embeddedpostgres.V16),
	)

	log.Printf("[embedded] starting postgres on port %d …", port)
	if err := pg.Start(); err != nil {
		return nil, fmt.Errorf("start embedded postgres: %w", err)
	}

	dsn := fmt.Sprintf("postgres://eol:eoldev@localhost:%d/eol?sslmode=disable", port)
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		pg.Stop()
		return nil, fmt.Errorf("connect to embedded postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		pg.Stop()
		return nil, fmt.Errorf("ping embedded postgres: %w", err)
	}
	log.Printf("[embedded] postgres ready on port %d", port)

	// ---- In-memory Redis (miniredis) ----------------------------------------
	mr, err := miniredis.Run()
	if err != nil {
		pool.Close()
		pg.Stop()
		return nil, fmt.Errorf("start miniredis: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	if err := rdb.Ping(ctx).Err(); err != nil {
		mr.Close()
		pool.Close()
		pg.Stop()
		return nil, fmt.Errorf("ping miniredis: %w", err)
	}
	log.Printf("[embedded] redis ready on %s", mr.Addr())

	return &EmbeddedDatabases{
		pg:      pg,
		minired: mr,
		Pool:    pool,
		Redis:   rdb,
	}, nil
}

// Stop gracefully shuts down both embedded databases.
func (e *EmbeddedDatabases) Stop() {
	if e.Redis != nil {
		e.Redis.Close()
	}
	if e.minired != nil {
		e.minired.Close()
	}
	if e.Pool != nil {
		e.Pool.Close()
	}
	if e.pg != nil {
		if err := e.pg.Stop(); err != nil {
			log.Printf("[embedded] postgres stop error: %v", err)
		}
	}
	log.Println("[embedded] databases stopped")
}

// freePort asks the OS for an available TCP port.
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}