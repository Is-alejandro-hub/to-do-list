package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/Is-alejandro-hub/to-do-list/backend/internal/domain"
)

type taskService struct {
	repo domain.TaskRepository
}

// NewTaskService construye el servicio. Recibe la interfaz del repositorio,
// así en tests se puede inyectar un mock sin tocar PostgreSQL.
func NewTaskService(repo domain.TaskRepository) TaskService {
	return &taskService{repo: repo}
}

func (s *taskService) Create(
	ctx context.Context, input domain.CreateTaskInput,
) (*domain.Task, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Regla de negocio: si no viene prioridad, aplicamos MEDIUM por defecto.
	// El dominio no fuerza el default para que el cliente pueda reusar
	// el mismo struct en otros contextos.
	if input.Priority == "" {
		input.Priority = domain.PriorityMedium
	}

	task := &domain.Task{
		Title:       strings.TrimSpace(input.Title),
		Description: strings.TrimSpace(input.Description),
		Priority:    input.Priority,
		DueDate:     input.DueDate,
		Completed:   false,
		Tags:        tagsFromNames(input.TagNames),
	}

	if err := s.repo.Create(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *taskService) GetByID(
	ctx context.Context, id uuid.UUID, includeDeleted bool,
) (*domain.Task, error) {
	return s.repo.GetByID(ctx, id, includeDeleted)
}

func (s *taskService) List(
	ctx context.Context, filter domain.TaskFilter,
) ([]domain.Task, error) {
	// Regla de negocio: el search se limpia antes de pasarlo al repo.
	// Evita consultas con espacios accidentales tipo "  hola  ".
	filter.Search = strings.TrimSpace(filter.Search)
	return s.repo.List(ctx, filter)
}

func (s *taskService) Update(
	ctx context.Context, id uuid.UUID, input domain.UpdateTaskInput,
) (*domain.Task, error) {
	if input.IsEmpty() {
		return nil, domain.NewValidationError("body", "no se envió ningún campo para actualizar")
	}
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Normalizamos strings en el servicio para no ensuciar el repositorio.
	if input.Title != nil {
		trimmed := strings.TrimSpace(*input.Title)
		input.Title = &trimmed
	}
	if input.Description != nil {
		trimmed := strings.TrimSpace(*input.Description)
		input.Description = &trimmed
	}

	return s.repo.Update(ctx, id, input)
}

func (s *taskService) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, id)
}

func (s *taskService) Restore(ctx context.Context, id uuid.UUID) error {
	return s.repo.Restore(ctx, id)
}

func (s *taskService) ListTags(ctx context.Context) ([]domain.Tag, error) {
	return s.repo.ListTags(ctx)
}

// tagsFromNames convierte una lista de nombres a Tags con solo el nombre
// poblado. El repositorio se encarga de resolver/crear el ID.
func tagsFromNames(names []string) []domain.Tag {
	if len(names) == 0 {
		return nil
	}
	// Deduplicamos conservando el orden. Si el cliente envía "trabajo"
	// dos veces, no queremos insertarlo dos veces.
	seen := make(map[string]struct{}, len(names))
	result := make([]domain.Tag, 0, len(names))
	for _, n := range names {
		trimmed := strings.TrimSpace(n)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, domain.Tag{Name: trimmed})
	}
	return result
}
func (s *taskService) ListAudit(
	ctx context.Context, taskID uuid.UUID,
) ([]domain.AuditEntry, error) {
	// Verificamos que la tarea existe (incluyendo eliminadas) antes de
	// devolver la auditoría. Si no existe, 404.
	if _, err := s.repo.GetByID(ctx, taskID, true); err != nil {
		return nil, err
	}
	return s.repo.ListAuditByTaskID(ctx, taskID)
}
