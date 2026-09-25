package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/Is-alejandro-hub/to-do-list/backend/internal/domain"
)

// mockTaskRepository implementa domain.TaskRepository.
// Cada método tiene un campo `fn` configurable. Si es nil, devuelve
// un default razonable (nil error, slice vacío, etc.).
type mockTaskRepository struct {
	CreateFn            func(ctx context.Context, task *domain.Task) error
	GetByIDFn           func(ctx context.Context, id uuid.UUID, includeDeleted bool) (*domain.Task, error)
	ListFn              func(ctx context.Context, filter domain.TaskFilter) ([]domain.Task, error)
	UpdateFn            func(ctx context.Context, id uuid.UUID, input domain.UpdateTaskInput) (*domain.Task, error)
	SoftDeleteFn        func(ctx context.Context, id uuid.UUID) error
	RestoreFn           func(ctx context.Context, id uuid.UUID) error
	ListTagsFn          func(ctx context.Context) ([]domain.Tag, error)
	ListAuditByTaskIDFn func(ctx context.Context, taskID uuid.UUID) ([]domain.AuditEntry, error)

	// Registro de llamadas para verificar desde el test.
	CreateCalls     int
	GetByIDCalls    int
	ListCalls       int
	UpdateCalls     int
	SoftDeleteCalls int
	RestoreCalls    int
	ListTagsCalls   int
	ListAuditCalls  int

	// Último input recibido, útil para verificar transformaciones.
	LastFilter domain.TaskFilter
	LastUpdate domain.UpdateTaskInput
}

func (m *mockTaskRepository) Create(ctx context.Context, task *domain.Task) error {
	m.CreateCalls++
	if m.CreateFn != nil {
		return m.CreateFn(ctx, task)
	}
	return nil
}

func (m *mockTaskRepository) GetByID(ctx context.Context, id uuid.UUID, includeDeleted bool) (*domain.Task, error) {
	m.GetByIDCalls++
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id, includeDeleted)
	}
	return nil, nil
}

func (m *mockTaskRepository) List(ctx context.Context, filter domain.TaskFilter) ([]domain.Task, error) {
	m.ListCalls++
	m.LastFilter = filter
	if m.ListFn != nil {
		return m.ListFn(ctx, filter)
	}
	return []domain.Task{}, nil
}

func (m *mockTaskRepository) Update(ctx context.Context, id uuid.UUID, input domain.UpdateTaskInput) (*domain.Task, error) {
	m.UpdateCalls++
	m.LastUpdate = input
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, id, input)
	}
	return nil, nil
}

func (m *mockTaskRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	m.SoftDeleteCalls++
	if m.SoftDeleteFn != nil {
		return m.SoftDeleteFn(ctx, id)
	}
	return nil
}

func (m *mockTaskRepository) Restore(ctx context.Context, id uuid.UUID) error {
	m.RestoreCalls++
	if m.RestoreFn != nil {
		return m.RestoreFn(ctx, id)
	}
	return nil
}

func (m *mockTaskRepository) ListTags(ctx context.Context) ([]domain.Tag, error) {
	m.ListTagsCalls++
	if m.ListTagsFn != nil {
		return m.ListTagsFn(ctx)
	}
	return []domain.Tag{}, nil
}

func (m *mockTaskRepository) ListAuditByTaskID(ctx context.Context, taskID uuid.UUID) ([]domain.AuditEntry, error) {
	m.ListAuditCalls++
	if m.ListAuditByTaskIDFn != nil {
		return m.ListAuditByTaskIDFn(ctx, taskID)
	}
	return []domain.AuditEntry{}, nil
}

// mockIdempotencyRepository implementa domain.IdempotencyRepository.
type mockIdempotencyRepository struct {
	FindFn          func(ctx context.Context, key uuid.UUID) (*domain.IdempotencyRecord, error)
	SaveFn          func(ctx context.Context, rec *domain.IdempotencyRecord) error
	DeleteExpiredFn func(ctx context.Context) (int64, error)

	FindCalls          int
	SaveCalls          int
	DeleteExpiredCalls int

	LastSaved *domain.IdempotencyRecord
}

func (m *mockIdempotencyRepository) Find(ctx context.Context, key uuid.UUID) (*domain.IdempotencyRecord, error) {
	m.FindCalls++
	if m.FindFn != nil {
		return m.FindFn(ctx, key)
	}
	return nil, nil
}

func (m *mockIdempotencyRepository) Save(ctx context.Context, rec *domain.IdempotencyRecord) error {
	m.SaveCalls++
	m.LastSaved = rec
	if m.SaveFn != nil {
		return m.SaveFn(ctx, rec)
	}
	return nil
}

func (m *mockIdempotencyRepository) DeleteExpired(ctx context.Context) (int64, error) {
	m.DeleteExpiredCalls++
	if m.DeleteExpiredFn != nil {
		return m.DeleteExpiredFn(ctx)
	}
	return 0, nil
}
