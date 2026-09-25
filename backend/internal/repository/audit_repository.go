package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/Is-alejandro-hub/to-do-list/backend/internal/domain"
)

// ListAuditByTaskID devuelve el historial de una tarea.
//
// El método vive en el mismo archivo que TaskRepository (o en uno propio)
// y se añade a PostgresTaskRepository. La firma coincide con la definida
// en domain.TaskRepository.
//
// Nota sobre JSONB: pgx escanea las columnas JSONB a []byte; nosotros las
// deserializamos a map[string]any. Si un campo es NULL en la base,
// el []byte viene vacío y dejamos el mapa como nil.
func (r *PostgresTaskRepository) ListAuditByTaskID(
	ctx context.Context, taskID uuid.UUID,
) ([]domain.AuditEntry, error) {
	const q = `
		SELECT id, task_id, action, old_data, new_data, changed_by, changed_at
		FROM task_audit_log
		WHERE task_id = $1
		ORDER BY changed_at DESC, id DESC
	`

	rows, err := r.pool.Query(ctx, q, taskID)
	if err != nil {
		return nil, fmt.Errorf("consultando auditoría: %w", err)
	}
	defer rows.Close()

	entries := []domain.AuditEntry{}
	for rows.Next() {
		var e domain.AuditEntry
		var oldRaw, newRaw []byte
		var action string

		if err := rows.Scan(
			&e.ID, &e.TaskID, &action, &oldRaw, &newRaw,
			&e.ChangedBy, &e.ChangedAt,
		); err != nil {
			return nil, fmt.Errorf("escaneando entrada de auditoría: %w", err)
		}
		e.Action = domain.AuditAction(action)

		// Deserializamos los JSONB a mapas.
		if len(oldRaw) > 0 {
			if err := json.Unmarshal(oldRaw, &e.OldData); err != nil {
				return nil, fmt.Errorf("deserializando old_data: %w", err)
			}
		}
		if len(newRaw) > 0 {
			if err := json.Unmarshal(newRaw, &e.NewData); err != nil {
				return nil, fmt.Errorf("deserializando new_data: %w", err)
			}
		}

		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterando auditoría: %w", err)
	}
	return entries, nil
}
