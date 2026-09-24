package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Is-alejandro-hub/to-do-list/backend/internal/domain"
)

// ============================================================================
// Create
// ============================================================================

func TestTaskService_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("éxito con datos mínimos aplica prioridad por defecto", func(t *testing.T) {
		repo := &mockTaskRepository{
			CreateFn: func(_ context.Context, task *domain.Task) error {
				task.ID = uuid.New()
				return nil
			},
		}
		svc := NewTaskService(repo)

		task, err := svc.Create(ctx, domain.CreateTaskInput{Title: "Hacer ejercicio"})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if task.Priority != domain.PriorityMedium {
			t.Errorf("Priority = %q, se esperaba MEDIUM por defecto", task.Priority)
		}
		if task.Completed {
			t.Errorf("Completed = true, se esperaba false por defecto")
		}
		if repo.CreateCalls != 1 {
			t.Errorf("CreateCalls = %d, se esperaba 1", repo.CreateCalls)
		}
	})

	t.Run("valida el input antes de llamar al repo", func(t *testing.T) {
		repo := &mockTaskRepository{}
		svc := NewTaskService(repo)

		_, err := svc.Create(ctx, domain.CreateTaskInput{Title: ""})
		if err == nil {
			t.Fatalf("se esperaba error de validación")
		}
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("errors.Is(err, ErrInvalidInput) = false")
		}
		if repo.CreateCalls != 0 {
			t.Errorf("CreateCalls = %d, se esperaba 0 (no debe llamar al repo)", repo.CreateCalls)
		}
	})

	t.Run("normaliza título con espacios", func(t *testing.T) {
		var capturedTitle string
		repo := &mockTaskRepository{
			CreateFn: func(_ context.Context, task *domain.Task) error {
				capturedTitle = task.Title
				task.ID = uuid.New()
				return nil
			},
		}
		svc := NewTaskService(repo)

		_, err := svc.Create(ctx, domain.CreateTaskInput{Title: "  Hacer ejercicio  "})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if capturedTitle != "Hacer ejercicio" {
			t.Errorf("título guardado = %q, se esperaba sin espacios", capturedTitle)
		}
	})

	t.Run("deduplica tags ignorando mayúsculas y espacios", func(t *testing.T) {
		var capturedTags []domain.Tag
		repo := &mockTaskRepository{
			CreateFn: func(_ context.Context, task *domain.Task) error {
				capturedTags = task.Tags
				task.ID = uuid.New()
				return nil
			},
		}
		svc := NewTaskService(repo)

		input := domain.CreateTaskInput{
			Title:    "X",
			TagNames: []string{"trabajo", "Trabajo", " TRABAJO ", "urgente"},
		}
		_, err := svc.Create(ctx, input)
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if len(capturedTags) != 2 {
			t.Fatalf("se esperaban 2 tags tras deduplicar, se obtuvieron %d", len(capturedTags))
		}
	})

	t.Run("propaga error del repositorio", func(t *testing.T) {
		repo := &mockTaskRepository{
			CreateFn: func(_ context.Context, _ *domain.Task) error {
				return errors.New("fallo de base de datos")
			},
		}
		svc := NewTaskService(repo)

		_, err := svc.Create(ctx, domain.CreateTaskInput{Title: "X"})
		if err == nil {
			t.Fatalf("se esperaba error del repositorio")
		}
	})
}

// ============================================================================
// List
// ============================================================================

func TestTaskService_List_TrimsSearch(t *testing.T) {
	ctx := context.Background()
	repo := &mockTaskRepository{}
	svc := NewTaskService(repo)

	_, err := svc.List(ctx, domain.TaskFilter{Search: "  ejercicio  "})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if repo.LastFilter.Search != "ejercicio" {
		t.Errorf("Search = %q, se esperaba sin espacios", repo.LastFilter.Search)
	}
}

func TestTaskService_List_PassesFilterToRepo(t *testing.T) {
	ctx := context.Background()
	repo := &mockTaskRepository{}
	svc := NewTaskService(repo)

	completed := true
	priority := domain.PriorityHigh
	input := domain.TaskFilter{
		Completed: &completed,
		Priority:  &priority,
	}

	_, err := svc.List(ctx, input)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if repo.LastFilter.Completed == nil || *repo.LastFilter.Completed != true {
		t.Errorf("Completed no se propagó correctamente")
	}
	if repo.LastFilter.Priority == nil || *repo.LastFilter.Priority != domain.PriorityHigh {
		t.Errorf("Priority no se propagó correctamente")
	}
}

// ============================================================================
// Update
// ============================================================================

func TestTaskService_Update(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()

	t.Run("rechaza update vacío sin llamar al repo", func(t *testing.T) {
		repo := &mockTaskRepository{}
		svc := NewTaskService(repo)

		_, err := svc.Update(ctx, id, domain.UpdateTaskInput{})
		if err == nil {
			t.Fatalf("se esperaba error por update vacío")
		}
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("errors.Is(err, ErrInvalidInput) = false")
		}
		if repo.UpdateCalls != 0 {
			t.Errorf("UpdateCalls = %d, se esperaba 0", repo.UpdateCalls)
		}
	})

	t.Run("normaliza título antes de llamar al repo", func(t *testing.T) {
		repo := &mockTaskRepository{
			UpdateFn: func(_ context.Context, _ uuid.UUID, _ domain.UpdateTaskInput) (*domain.Task, error) {
				return &domain.Task{}, nil
			},
		}
		svc := NewTaskService(repo)

		title := "  Nuevo título  "
		_, err := svc.Update(ctx, id, domain.UpdateTaskInput{Title: &title})
		if err != nil {
			t.Fatalf("error inesperado: %v", err)
		}
		if repo.LastUpdate.Title == nil || *repo.LastUpdate.Title != "Nuevo título" {
			t.Errorf("título no normalizado correctamente: %v", repo.LastUpdate.Title)
		}
	})

	t.Run("propaga ErrTaskNotFound del repositorio", func(t *testing.T) {
		repo := &mockTaskRepository{
			UpdateFn: func(_ context.Context, _ uuid.UUID, _ domain.UpdateTaskInput) (*domain.Task, error) {
				return nil, domain.ErrTaskNotFound
			},
		}
		svc := NewTaskService(repo)

		title := "X"
		_, err := svc.Update(ctx, id, domain.UpdateTaskInput{Title: &title})
		if !errors.Is(err, domain.ErrTaskNotFound) {
			t.Errorf("errors.Is(err, ErrTaskNotFound) = false, err = %v", err)
		}
	})
}

// ============================================================================
// SoftDelete / Restore / ListTags (delegación pura)
// ============================================================================

func TestTaskService_SoftDelete_Delegates(t *testing.T) {
	repo := &mockTaskRepository{}
	svc := NewTaskService(repo)

	if err := svc.SoftDelete(context.Background(), uuid.New()); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if repo.SoftDeleteCalls != 1 {
		t.Errorf("SoftDeleteCalls = %d, se esperaba 1", repo.SoftDeleteCalls)
	}
}

func TestTaskService_Restore_Delegates(t *testing.T) {
	repo := &mockTaskRepository{}
	svc := NewTaskService(repo)

	if err := svc.Restore(context.Background(), uuid.New()); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if repo.RestoreCalls != 1 {
		t.Errorf("RestoreCalls = %d, se esperaba 1", repo.RestoreCalls)
	}
}

func TestTaskService_ListTags_Delegates(t *testing.T) {
	repo := &mockTaskRepository{}
	svc := NewTaskService(repo)

	_, err := svc.ListTags(context.Background())
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if repo.ListTagsCalls != 1 {
		t.Errorf("ListTagsCalls = %d, se esperaba 1", repo.ListTagsCalls)
	}
}

// ============================================================================
// tagsFromNames (función interna)
// ============================================================================

func TestTagsFromNames(t *testing.T) {
	cases := []struct {
		name  string
		input []string
		want  int
	}{
		{"nil", nil, 0},
		{"slice vacío", []string{}, 0},
		{"un tag", []string{"trabajo"}, 1},
		{"dos tags distintos", []string{"trabajo", "urgente"}, 2},
		{"duplicados exactos", []string{"trabajo", "trabajo"}, 1},
		{"duplicados con mayúsculas", []string{"trabajo", "TRABAJO"}, 1},
		{"duplicados con espacios", []string{"trabajo", "  trabajo  "}, 1},
		{"ignora vacíos", []string{"trabajo", "", "   "}, 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tagsFromNames(tc.input)
			if len(got) != tc.want {
				t.Errorf("len(tagsFromNames(%v)) = %d, se esperaba %d",
					tc.input, len(got), tc.want)
			}
		})
	}
}
