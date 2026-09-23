package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// TIPOS BÁSICOS
// ============================================================================

// Priority representa el nivel de prioridad de una tarea.
// Es un tipo propio (no un string suelto) para que el compilador nos ayude:
// no podemos pasar un string cualquiera donde se espera un Priority.
type Priority string

const (
	PriorityLow    Priority = "LOW"
	PriorityMedium Priority = "MEDIUM"
	PriorityHigh   Priority = "HIGH"
)

// IsValid comprueba si el valor es uno de los permitidos.
// Debe mantenerse sincronizado con el ENUM de PostgreSQL.
func (p Priority) IsValid() bool {
	switch p {
	case PriorityLow, PriorityMedium, PriorityHigh:
		return true
	}
	return false
}

// ============================================================================
// ENTIDADES
// ============================================================================

// Tag es una etiqueta reutilizable.
type Tag struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// Task es la entidad central del sistema.
//
// Los tags se cargan por separado cuando se necesitan (no en cada GET),
// para evitar joins innecesarios. El repositorio decide si hidratarlos
// o no según la operación.
type Task struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Priority    Priority   `json:"priority"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	Completed   bool       `json:"completed"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Tags        []Tag      `json:"tags,omitempty"`
}

// IsDeleted es un helper de lectura rápida.
func (t Task) IsDeleted() bool {
	return t.DeletedAt != nil
}

// ============================================================================
// DTOs DE ENTRADA
// ============================================================================

// CreateTaskInput es lo que el cliente envía para crear una tarea.
//
// Nota: NO incluye ID (lo genera la BD), ni timestamps (los pone la BD),
// ni Completed (siempre arranca en false), ni DeletedAt (nunca se crea
// una tarea ya eliminada). El principio es: el cliente solo envía lo
// que puede controlar.
type CreateTaskInput struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Priority    Priority   `json:"priority"`
	DueDate     *time.Time `json:"due_date"`
	TagNames    []string   `json:"tag_names"`
}

// UpdateTaskInput es lo que el cliente envía para editar una tarea.
//
// Usamos punteros para cada campo. ¿Por qué? Porque en Go, el "valor cero"
// de un string es "" y el de un bool es false. Si usáramos valores planos,
// no podríamos distinguir entre:
//   - "el cliente no envió el campo completed" (debe quedarse como estaba)
//   - "el cliente envió completed=false explícitamente" (debe actualizarse)
//
// Con punteros, nil significa "no enviado" y un valor real significa
// "actualiza a este valor". Es un patrón estándar en APIs PATCH.
type UpdateTaskInput struct {
	Title       *string    `json:"title,omitempty"`
	Description *string    `json:"description,omitempty"`
	Priority    *Priority  `json:"priority,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	Completed   *bool      `json:"completed,omitempty"`
	TagNames    *[]string  `json:"tag_names,omitempty"`
}

// TaskFilter agrupa los criterios de búsqueda del listado.
//
// Todos son punteros o slices porque son opcionales. El repositorio
// construye la query dinámicamente según cuáles vengan.
type TaskFilter struct {
	IncludeDeleted *bool      `json:"include_deleted,omitempty"`
	Completed      *bool      `json:"completed,omitempty"`
	Priority       *Priority  `json:"priority,omitempty"`
	DueBefore      *time.Time `json:"due_before,omitempty"`
	DueAfter       *time.Time `json:"due_after,omitempty"`
	TagNames       []string   `json:"tag_names,omitempty"`
	Search         string     `json:"search,omitempty"`
}

// ============================================================================
// VALIDACIÓN
// ============================================================================

// Validate comprueba que el input de creación sea correcto.
// Devuelve *ValidationError (que envuelve a ErrInvalidInput) o nil.
func (in CreateTaskInput) Validate() error {
	trimmed := strings.TrimSpace(in.Title)
	if trimmed == "" {
		return NewValidationError("title", "el título no puede estar vacío")
	}
	if len(trimmed) > 255 {
		return NewValidationError("title", "el título no puede superar los 255 caracteres")
	}

	if in.Priority == "" {
		// El servicio lo puede rellenar con PriorityMedium por defecto;
		// aquí solo rechazamos valores que no son válidos.
	} else if !in.Priority.IsValid() {
		return NewValidationError("priority", "prioridad inválida (LOW, MEDIUM o HIGH)")
	}

	for _, tag := range in.TagNames {
		if strings.TrimSpace(tag) == "" {
			return NewValidationError("tag_names", "las etiquetas no pueden estar vacías")
		}
		if len(tag) > 50 {
			return NewValidationError("tag_names", "cada etiqueta puede tener máximo 50 caracteres")
		}
	}

	return nil
}

// Validate comprueba el input de actualización. Al ser todos los campos
// opcionales, solo validamos los que vienen.
func (in UpdateTaskInput) Validate() error {
	if in.Title != nil {
		trimmed := strings.TrimSpace(*in.Title)
		if trimmed == "" {
			return NewValidationError("title", "el título no puede estar vacío")
		}
		if len(trimmed) > 255 {
			return NewValidationError("title", "el título no puede superar los 255 caracteres")
		}
	}

	if in.Priority != nil && !in.Priority.IsValid() {
		return NewValidationError("priority", "prioridad inválida (LOW, MEDIUM o HIGH)")
	}

	if in.TagNames != nil {
		for _, tag := range *in.TagNames {
			if strings.TrimSpace(tag) == "" {
				return NewValidationError("tag_names", "las etiquetas no pueden estar vacías")
			}
			if len(tag) > 50 {
				return NewValidationError("tag_names", "cada etiqueta puede tener máximo 50 caracteres")
			}
		}
	}

	return nil
}

// IsEmpty comprueba si un UpdateTaskInput no tiene ningún campo.
// El servicio puede rechazar updates vacíos con un 400.
func (in UpdateTaskInput) IsEmpty() bool {
	return in.Title == nil &&
		in.Description == nil &&
		in.Priority == nil &&
		in.DueDate == nil &&
		in.Completed == nil &&
		in.TagNames == nil
}
