package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Is-alejandro-hub/to-do-list/backend/internal/config"
)

// NewPool crea un pool de conexiones a PostgreSQL.
//
// Usamos pgxpool (no pgx.Conn) porque un servidor HTTP atiende múltiples
// requests concurrentes. Abrir una conexión nueva por request es carísimo;
// el pool reutiliza conexiones.
func NewPool(ctx context.Context, cfg config.DBConfig) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("error parseando DSN: %w", err)
	}

	// Ajustes del pool. Los valores son razonables para desarrollo.
	// En producción se calculan según max_connections del servidor PostgreSQL.
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("error creando pool: %w", err)
	}

	// Ping con timeout para no colgar el arranque si la BD no responde.
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("no se pudo conectar a PostgreSQL: %w", err)
	}

	return pool, nil
}
