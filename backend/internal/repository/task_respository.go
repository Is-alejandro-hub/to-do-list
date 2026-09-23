package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Is-alejandro-hub/to-do-list/backend/internal/domain"
)

// PostgresTaskRepository implementa domain.TaskRepository sobre PostgreSQL.
type PostgresTaskRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresTaskRepository es el constructor. Devuelve la interfaz, no
// el tipo concreto, siguiendo el principio "accept interfaces, return structs"
// invertido: devolvemos interfaz cuando queremos ocultar la implementación.
func NewPostgresTaskRepository(pool *pgxpool.Pool) domain.TaskRepository {
	return &PostgresTaskRepository{pool: pool}
}

// ============================================================================
// CREATE
// ============================================================================

func (r *PostgresTaskRepository) Create(ctx context.Context, task *domain.Task) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("iniciando transacción: %w", err)
	}
	// Rollback diferido. Si el commit tiene éxito, este rollback es no-op.
	// Si algo falla antes del commit, deshace todo.
	defer func() { _ = tx.Rollback(ctx) }()

	const insertTask = `
		INSERT INTO tasks (title, description, priority, due_date)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	err = tx.QueryRow(ctx, insertTask,
		task.Title,
		task.Description,
		string(task.Priority),
		task.DueDate,
	).Scan(&task.ID, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insertando tarea: %w", err)
	}

	// Hidratamos los tags (si vienen) dentro de la misma transacción.
	if len(task.Tags) > 0 {
		if err := r.upsertAndLinkTags(ctx, tx, task.ID, task.Tags); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// upsertAndLinkTags recibe un []Tag porque el Create inicialmente los trae
// con solo Name poblado. Este método los persiste y rellena el ID.
func (r *PostgresTaskRepository) upsertAndLinkTags(
	ctx context.Context, tx pgx.Tx, taskID uuid.UUID, tags []domain.Tag,
) error {
	for i := range tags {
		tagID, err := r.upsertTag(ctx, tx, tags[i].Name)
		if err != nil {
			return err
		}
		tags[i].ID = tagID

		if _, err := tx.Exec(ctx,
			`INSERT INTO task_tags (task_id, tag_id) VALUES ($1, $2)
			 ON CONFLICT DO NOTHING`,
			taskID, tagID,
		); err != nil {
			return fmt.Errorf("vinculando tag %q: %w", tags[i].Name, err)
		}
	}
	return nil
}

// upsertTag inserta un tag si no existe, o recupera el existente si ya está.
// El "DO UPDATE SET name = EXCLUDED.name" es el truco estándar para forzar
// que RETURNING devuelva la fila incluso cuando hay conflicto.
func (r *PostgresTaskRepository) upsertTag(
	ctx context.Context, tx pgx.Tx, name string,
) (uuid.UUID, error) {
	const q = `
		INSERT INTO tags (name) VALUES ($1)
		ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`
	var id uuid.UUID
	if err := tx.QueryRow(ctx, q, strings.TrimSpace(name)).Scan(&id); err != nil {
		return uuid.Nil, fmt.Errorf("upsert tag %q: %w", name, err)
	}
	return id, nil
}

// ============================================================================
// GET BY ID
// ============================================================================

func (r *PostgresTaskRepository) GetByID(
	ctx context.Context, id uuid.UUID, includeDeleted bool,
) (*domain.Task, error) {
	query := `
		SELECT id, title, COALESCE(description, ''), priority,
		       due_date, completed, deleted_at, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`
	if !includeDeleted {
		query += ` AND deleted_at IS NULL`
	}

	var t domain.Task
	var priority string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.Title, &t.Description, &priority,
		&t.DueDate, &t.Completed, &t.DeletedAt, &t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrTaskNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("consultando tarea: %w", err)
	}
	t.Priority = domain.Priority(priority)

	// Cargamos tags en una segunda query (evita multiplicar filas por JOIN).
	tags, err := r.loadTagsForTasks(ctx, []uuid.UUID{t.ID})
	if err != nil {
		return nil, err
	}
	t.Tags = tags[t.ID]
	if t.Tags == nil {
		t.Tags = []domain.Tag{}
	}

	return &t, nil
}

// loadTagsForTasks devuelve un mapa task_id → tags. Una sola query para N tareas.
func (r *PostgresTaskRepository) loadTagsForTasks(
	ctx context.Context, taskIDs []uuid.UUID,
) (map[uuid.UUID][]domain.Tag, error) {
	if len(taskIDs) == 0 {
		return map[uuid.UUID][]domain.Tag{}, nil
	}

	const q = `
		SELECT tt.task_id, t.id, t.name, t.created_at
		FROM task_tags tt
		JOIN tags t ON t.id = tt.tag_id
		WHERE tt.task_id = ANY($1)
		ORDER BY t.name
	`

	rows, err := r.pool.Query(ctx, q, taskIDs)
	if err != nil {
		return nil, fmt.Errorf("cargando tags: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]domain.Tag)
	for rows.Next() {
		var taskID uuid.UUID
		var tag domain.Tag
		if err := rows.Scan(&taskID, &tag.ID, &tag.Name, &tag.CreatedAt); err != nil {
			return nil, fmt.Errorf("escaneando tag: %w", err)
		}
		result[taskID] = append(result[taskID], tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterando tags: %w", err)
	}
	return result, nil
}

// ============================================================================
// LIST
// ============================================================================

func (r *PostgresTaskRepository) List(
	ctx context.Context, filter domain.TaskFilter,
) ([]domain.Task, error) {
	// Construcción dinámica: acumulamos condiciones y argumentos.
	// Usamos un contador explícito porque pgx numera los placeholders
	// posicionalmente ($1, $2, ...) y el orden importa.
	conditions := []string{"1=1"}
	args := []any{}
	argIdx := 1

	// Soft delete: por defecto excluimos eliminadas.
	if filter.IncludeDeleted == nil || !*filter.IncludeDeleted {
		conditions = append(conditions, "deleted_at IS NULL")
	}

	if filter.Completed != nil {
		conditions = append(conditions, fmt.Sprintf("completed = $%d", argIdx))
		args = append(args, *filter.Completed)
		argIdx++
	}

	if filter.Priority != nil {
		conditions = append(conditions, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, string(*filter.Priority))
		argIdx++
	}

	if filter.DueBefore != nil {
		conditions = append(conditions, fmt.Sprintf("due_date <= $%d", argIdx))
		args = append(args, *filter.DueBefore)
		argIdx++
	}

	if filter.DueAfter != nil {
		conditions = append(conditions, fmt.Sprintf("due_date >= $%d", argIdx))
		args = append(args, *filter.DueAfter)
		argIdx++
	}

	if filter.Search != "" {
		// ILIKE para búsqueda case-insensitive. El % se añade al valor,
		// no al SQL, para evitar inyección.
		conditions = append(conditions,
			fmt.Sprintf("(title ILIKE $%d OR COALESCE(description, '') ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	if len(filter.TagNames) > 0 {
		// EXISTS es más eficiente que JOIN + DISTINCT cuando solo
		// queremos filtrar, no traer columnas del tag.
		conditions = append(conditions, fmt.Sprintf(`
			EXISTS (
				SELECT 1 FROM task_tags tt
				JOIN tags tg ON tg.id = tt.tag_id
				WHERE tt.task_id = tasks.id AND tg.name = ANY($%d)
			)`, argIdx))
		args = append(args, filter.TagNames)
		argIdx++
	}

	query := `
		SELECT id, title, COALESCE(description, ''), priority,
		       due_date, completed, deleted_at, created_at, updated_at
		FROM tasks
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY
		    CASE priority WHEN 'HIGH' THEN 1 WHEN 'MEDIUM' THEN 2 ELSE 3 END,
		    due_date NULLS LAST,
		    created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("consultando tareas: %w", err)
	}
	defer rows.Close()

	tasks := []domain.Task{} // Inicializamos vacío, no nil, para JSON [].
	ids := []uuid.UUID{}
	for rows.Next() {
		var t domain.Task
		var priority string
		if err := rows.Scan(
			&t.ID, &t.Title, &t.Description, &priority,
			&t.DueDate, &t.Completed, &t.DeletedAt, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("escaneando tarea: %w", err)
		}
		t.Priority = domain.Priority(priority)
		tasks = append(tasks, t)
		ids = append(ids, t.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterando tareas: %w", err)
	}

	// Cargamos todos los tags en una sola query.
	tagsByTask, err := r.loadTagsForTasks(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range tasks {
		if tags := tagsByTask[tasks[i].ID]; tags != nil {
			tasks[i].Tags = tags
		} else {
			tasks[i].Tags = []domain.Tag{}
		}
	}

	return tasks, nil
}

// ============================================================================
// UPDATE
// ============================================================================

func (r *PostgresTaskRepository) Update(
	ctx context.Context, id uuid.UUID, input domain.UpdateTaskInput,
) (*domain.Task, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("iniciando transacción: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Verificamos existencia con lock. FOR UPDATE evita condiciones de carrera
	// (dos updates simultáneos sobre la misma fila).
	var exists bool
	err = tx.QueryRow(ctx,
		`SELECT TRUE FROM tasks WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`,
		id,
	).Scan(&exists)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrTaskNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("verificando tarea: %w", err)
	}

	// SET dinámico con placeholders posicionales.
	sets := []string{}
	args := []any{}
	argIdx := 1

	if input.Title != nil {
		sets = append(sets, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, strings.TrimSpace(*input.Title))
		argIdx++
	}
	if input.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *input.Description)
		argIdx++
	}
	if input.Priority != nil {
		sets = append(sets, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, string(*input.Priority))
		argIdx++
	}
	if input.DueDate != nil {
		sets = append(sets, fmt.Sprintf("due_date = $%d", argIdx))
		args = append(args, *input.DueDate)
		argIdx++
	}
	if input.Completed != nil {
		sets = append(sets, fmt.Sprintf("completed = $%d", argIdx))
		args = append(args, *input.Completed)
		argIdx++
	}

	if len(sets) > 0 {
		args = append(args, id)
		query := fmt.Sprintf(
			`UPDATE tasks SET %s WHERE id = $%d`,
			strings.Join(sets, ", "), argIdx,
		)
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return nil, fmt.Errorf("actualizando tarea: %w", err)
		}
	}

	// Manejo de tags: si el cliente envió el campo, reemplazamos el set completo.
	if input.TagNames != nil {
		if _, err := tx.Exec(ctx,
			`DELETE FROM task_tags WHERE task_id = $1`, id,
		); err != nil {
			return nil, fmt.Errorf("limpiando tags: %w", err)
		}
		tags := make([]domain.Tag, len(*input.TagNames))
		for i, name := range *input.TagNames {
			tags[i] = domain.Tag{Name: name}
		}
		if err := r.upsertAndLinkTags(ctx, tx, id, tags); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	// Recargamos fuera de la transacción para devolver el estado final.
	return r.GetByID(ctx, id, false)
}

// ============================================================================
// SOFT DELETE Y RESTORE
// ============================================================================

func (r *PostgresTaskRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	// Idempotente: si ya estaba eliminada, no hace nada y no es error.
	tag, err := r.pool.Exec(ctx,
		`UPDATE tasks SET deleted_at = NOW()
		 WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("soft delete: %w", err)
	}

	if tag.RowsAffected() == 0 {
		// Distinguimos: ¿no existe la tarea, o ya estaba eliminada?
		var exists bool
		err := r.pool.QueryRow(ctx,
			`SELECT TRUE FROM tasks WHERE id = $1`, id).Scan(&exists)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrTaskNotFound
		}
		if err != nil {
			return fmt.Errorf("verificando tarea: %w", err)
		}
		// Existe y ya estaba eliminada → idempotente, OK.
	}
	return nil
}

func (r *PostgresTaskRepository) Restore(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE tasks SET deleted_at = NULL
		 WHERE id = $1 AND deleted_at IS NOT NULL`, id)
	if err != nil {
		return fmt.Errorf("restore: %w", err)
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		err := r.pool.QueryRow(ctx,
			`SELECT TRUE FROM tasks WHERE id = $1`, id).Scan(&exists)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrTaskNotFound
		}
		if err != nil {
			return fmt.Errorf("verificando tarea: %w", err)
		}
	}
	return nil
}

// ============================================================================
// TAGS (catálogo)
// ============================================================================

func (r *PostgresTaskRepository) ListTags(ctx context.Context) ([]domain.Tag, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, created_at FROM tags ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("listando tags: %w", err)
	}
	defer rows.Close()

	tags := []domain.Tag{}
	for rows.Next() {
		var t domain.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("escaneando tag: %w", err)
		}
		tags = append(tags, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterando tags: %w", err)
	}
	return tags, nil
}
