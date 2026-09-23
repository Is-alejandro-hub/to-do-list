package domain

import (
	"context"

	"github.com/google/uuid"
)

// TaskRepository define las operaciones de persistencia sobre tareas y tags.
//
// Vive en el dominio (no en la carpeta repository) para invertir la
// dependencia: el servicio depende de esta interfaz, no de pgx ni de
// PostgreSQL. En tests unitarios, se puede inyectar un mock.
type TaskRepository interface {
	// Create inserta la tarea junto con sus tags en una sola transacción.
	// Al terminar, el *Task pasado queda hidratado con ID y timestamps
	// generados por la base de datos.
	Create(ctx context.Context, task *Task) error

	// GetByID devuelve una tarea por su ID. Si includeDeleted es false,
	// devuelve ErrTaskNotFound para tareas con deleted_at != NULL.
	GetByID(ctx context.Context, id uuid.UUID, includeDeleted bool) (*Task, error)

	// List devuelve tareas según los filtros. Siempre devuelve slice,
	// nunca nil, para que el JSON sea [] y no null.
	List(ctx context.Context, filter TaskFilter) ([]Task, error)

	// Update aplica cambios parciales. Devuelve la tarea ya actualizada.
	Update(ctx context.Context, id uuid.UUID, input UpdateTaskInput) (*Task, error)

	// SoftDelete marca la tarea como eliminada. Idempotente: si ya estaba
	// eliminada, no falla.
	SoftDelete(ctx context.Context, id uuid.UUID) error

	// Restore revierte un soft delete.
	Restore(ctx context.Context, id uuid.UUID) error

	// ListTags devuelve el catálogo completo de tags ordenado por nombre.
	ListTags(ctx context.Context) ([]Tag, error)
}

// IdempotencyRepository gestiona las claves de idempotencia del POST.
type IdempotencyRepository interface {
	// Find devuelve el registro asociado a una key, o nil si no existe.
	Find(ctx context.Context, key uuid.UUID) (*IdempotencyRecord, error)

	// Save persiste el resultado de una operación exitosa.
	Save(ctx context.Context, record *IdempotencyRecord) error

	// DeleteExpired elimina registros cuya expires_at ya pasó.
	// Devuelve cuántos se borraron.
	DeleteExpired(ctx context.Context) (int64, error)
}
