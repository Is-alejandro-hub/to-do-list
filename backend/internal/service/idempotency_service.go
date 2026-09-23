package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Is-alejandro-hub/to-do-list/backend/internal/domain"
)

// idempotencyTTL es cuánto tiempo guardamos la respuesta de una operación.
// 24h cubre cualquier reintento razonable de red sin acumular basura.
// Stripe usa 24h; es el estándar de la industria.
const idempotencyTTL = 24 * time.Hour

type idempotencyService struct {
	repo domain.IdempotencyRepository
}

func NewIdempotencyService(repo domain.IdempotencyRepository) IdempotencyService {
	return &idempotencyService{repo: repo}
}

func (s *idempotencyService) Check(
	ctx context.Context, key uuid.UUID, requestHash string,
) (*domain.IdempotencyRecord, error) {
	rec, err := s.repo.Find(ctx, key)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, nil
	}
	if rec.RequestHash != requestHash {
		// La misma key con distinto body → conflicto.
		// Es un abuso del contrato de idempotencia.
		return nil, domain.ErrIdempotencyConflict
	}
	return rec, nil
}

func (s *idempotencyService) Save(
	ctx context.Context, key uuid.UUID, requestHash string,
	status int, body []byte,
) error {
	rec := &domain.IdempotencyRecord{
		Key:            key,
		RequestHash:    requestHash,
		ResponseStatus: status,
		ResponseBody:   body,
		ExpiresAt:      time.Now().Add(idempotencyTTL),
	}
	return s.repo.Save(ctx, rec)
}

func (s *idempotencyService) CleanupExpired(ctx context.Context) (int64, error) {
	return s.repo.DeleteExpired(ctx)
}
