package domain

import (
	"time"

	"github.com/google/uuid"
)

// AuditAction representa el tipo de operación registrada.
type AuditAction string

const (
	AuditActionInsert     AuditAction = "INSERT"
	AuditActionUpdate     AuditAction = "UPDATE"
	AuditActionSoftDelete AuditAction = "SOFT_DELETE"
	AuditActionRestore    AuditAction = "RESTORE"
	AuditActionDelete     AuditAction = "DELETE"
)

// AuditEntry es un registro del historial de cambios de una tarea.
// Refleja una fila de task_audit_log.
//
// Nota: OldData y NewData son mapas (no structs) porque representan
// snapshots completos de la tarea en un momento dado, con todos sus
// campos. Al ser JSONB en PostgreSQL, en Go los tratamos como
// map[string]any para máxima flexibilidad.
type AuditEntry struct {
	ID        int64          `json:"id"`
	TaskID    uuid.UUID      `json:"task_id"`
	Action    AuditAction    `json:"action"`
	OldData   map[string]any `json:"old_data,omitempty"`
	NewData   map[string]any `json:"new_data,omitempty"`
	ChangedBy string         `json:"changed_by"`
	ChangedAt time.Time      `json:"changed_at"`
}
