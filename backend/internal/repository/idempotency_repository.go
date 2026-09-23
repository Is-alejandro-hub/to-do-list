package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Is-alejandro-hub/to-do-list/backend/internal/domain"
)

type PostgresIdempotencyRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresIdempotencyRepository(pool *pgxpool.Pool) domain.IdempotencyRepository {
	return &PostgresIdempotencyRepository{pool: pool}
}

func (r *PostgresIdempotencyRepository) Find(
	ctx context.Context, key uuid.UUID,
) (*domain.IdempotencyRecord, error) {
	const q = `
		SELECT key, request_hash, response_status, response_body,
		       created_at, expires_at
		FROM idempotency_keys
		WHERE key = $1 AND expires_at > NOW()
	`
	var rec domain.IdempotencyRecord
	err := r.pool.QueryRow(ctx, q, key).Scan(
		&rec.Key, &rec.RequestHash, &rec.ResponseStatus, &rec.ResponseBody,
		&rec.CreatedAt, &rec.ExpiresAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // No existir no es error, es "nunca procesado".
	}
	if err != nil {
		return nil, fmt.Errorf("buscando clave de idempotencia: %w", err)
	}
	return &rec, nil
}

func (r *PostgresIdempotencyRepository) Save(
	ctx context.Context, rec *domain.IdempotencyRecord,
) error {
	const q = `
		INSERT INTO idempotency_keys
			(key, request_hash, response_status, response_body, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (key) DO NOTHING
	`
	_, err := r.pool.Exec(ctx, q,
		rec.Key, rec.RequestHash, rec.ResponseStatus,
		rec.ResponseBody, rec.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("guardando clave de idempotencia: %w", err)
	}
	return nil
}

func (r *PostgresIdempotencyRepository) DeleteExpired(ctx context.Context) (int64, error) {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM idempotency_keys WHERE expires_at <= NOW()`)
	if err != nil {
		return 0, fmt.Errorf("limpiando claves expiradas: %w", err)
	}
	return tag.RowsAffected(), nil
}
