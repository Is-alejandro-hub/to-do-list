package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Is-alejandro-hub/to-do-list/backend/internal/domain"
)

func TestIdempotencyService_Check(t *testing.T) {
	ctx := context.Background()
	key := uuid.New()
	hash := "hash-correcto"

	t.Run("clave nueva devuelve nil sin error", func(t *testing.T) {
		repo := &mockIdempotencyRepository{
			FindFn: func(_ context.Context, _ uuid.UUID) (*domain.IdempotencyRecord, error) {
				return nil, nil
			},
		}
		svc := NewIdempotencyService(repo)

		rec, err := svc.Check(ctx, key, hash)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if rec != nil {
			t.Errorf("se esperaba nil, se obtuvo %+v", rec)
		}
	})

	t.Run("clave existente con mismo hash devuelve record", func(t *testing.T) {
		stored := &domain.IdempotencyRecord{
			Key:            key,
			RequestHash:    hash,
			ResponseStatus: 201,
		}
		repo := &mockIdempotencyRepository{
			FindFn: func(_ context.Context, _ uuid.UUID) (*domain.IdempotencyRecord, error) {
				return stored, nil
			},
		}
		svc := NewIdempotencyService(repo)

		rec, err := svc.Check(ctx, key, hash)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if rec != stored {
			t.Errorf("se esperaba el mismo record almacenado")
		}
	})

	t.Run("clave existente con hash distinto devuelve conflicto", func(t *testing.T) {
		stored := &domain.IdempotencyRecord{
			Key:         key,
			RequestHash: "otro-hash",
		}
		repo := &mockIdempotencyRepository{
			FindFn: func(_ context.Context, _ uuid.UUID) (*domain.IdempotencyRecord, error) {
				return stored, nil
			},
		}
		svc := NewIdempotencyService(repo)

		_, err := svc.Check(ctx, key, hash)
		if !errors.Is(err, domain.ErrIdempotencyConflict) {
			t.Errorf("errors.Is(err, ErrIdempotencyConflict) = false, err = %v", err)
		}
	})

	t.Run("propaga error del repositorio", func(t *testing.T) {
		repo := &mockIdempotencyRepository{
			FindFn: func(_ context.Context, _ uuid.UUID) (*domain.IdempotencyRecord, error) {
				return nil, errors.New("fallo de BD")
			},
		}
		svc := NewIdempotencyService(repo)

		_, err := svc.Check(ctx, key, hash)
		if err == nil {
			t.Fatalf("se esperaba error del repositorio")
		}
	})
}

func TestIdempotencyService_Save(t *testing.T) {
	ctx := context.Background()
	key := uuid.New()
	hash := "abc123"
	body := []byte(`{"id":"..."}`)

	repo := &mockIdempotencyRepository{}
	svc := NewIdempotencyService(repo)

	err := svc.Save(ctx, key, hash, 201, body)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if repo.SaveCalls != 1 {
		t.Fatalf("SaveCalls = %d, se esperaba 1", repo.SaveCalls)
	}

	saved := repo.LastSaved
	if saved.Key != key {
		t.Errorf("Key = %v, se esperaba %v", saved.Key, key)
	}
	if saved.RequestHash != hash {
		t.Errorf("RequestHash = %q, se esperaba %q", saved.RequestHash, hash)
	}
	if saved.ResponseStatus != 201 {
		t.Errorf("ResponseStatus = %d, se esperaba 201", saved.ResponseStatus)
	}
	if string(saved.ResponseBody) != string(body) {
		t.Errorf("ResponseBody = %s, se esperaba %s", saved.ResponseBody, body)
	}
	if saved.ExpiresAt.Before(saved.CreatedAt) {
		t.Errorf("ExpiresAt debe ser posterior a CreatedAt")
	}
}

func TestIdempotencyService_CleanupExpired(t *testing.T) {
	repo := &mockIdempotencyRepository{
		DeleteExpiredFn: func(_ context.Context) (int64, error) {
			return 5, nil
		},
	}
	svc := NewIdempotencyService(repo)

	n, err := svc.CleanupExpired(context.Background())
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if n != 5 {
		t.Errorf("n = %d, se esperaba 5", n)
	}
	if repo.DeleteExpiredCalls != 1 {
		t.Errorf("DeleteExpiredCalls = %d, se esperaba 1", repo.DeleteExpiredCalls)
	}
}
