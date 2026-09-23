package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/Is-alejandro-hub/to-do-list/backend/internal/domain"
)

// TaskService expone las operaciones de negocio sobre tareas.
// El handler dependerá de esta interfaz, no del struct concreto.
type TaskService interface {
	Create(ctx context.Context, input domain.CreateTaskInput) (*domain.Task, error)
	GetByID(ctx context.Context, id uuid.UUID, includeDeleted bool) (*domain.Task, error)
	List(ctx context.Context, filter domain.TaskFilter) ([]domain.Task, error)
	Update(ctx context.Context, id uuid.UUID, input domain.UpdateTaskInput) (*domain.Task, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) error
	ListTags(ctx context.Context) ([]domain.Tag, error)
}

// IdempotencyService gestiona el ciclo de vida de las claves de idempotencia.
type IdempotencyService interface {
	// Check comprueba si una clave ya fue procesada.
	// Devuelve:
	//   - (record, nil) si existe y el hash coincide → respuesta cacheada.
	//   - (nil, nil) si no existe → hay que procesar la operación.
	//   - (nil, ErrIdempotencyConflict) si existe con hash distinto.
	Check(ctx context.Context, key uuid.UUID, requestHash string) (*domain.IdempotencyRecord, error)

	// Save persiste el resultado de una operación exitosa.
	Save(ctx context.Context, key uuid.UUID, requestHash string, status int, body []byte) error

	// CleanupExpired elimina registros vencidos. Se llamará periódicamente.
	CleanupExpired(ctx context.Context) (int64, error)
}
