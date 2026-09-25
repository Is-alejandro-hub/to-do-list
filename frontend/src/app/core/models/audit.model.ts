/**
 * Tipo de acción registrada en la auditoría.
 * Mapea 1:1 con el CHECK constraint de la tabla task_audit_log.
 */
export type AuditAction = 'INSERT' | 'UPDATE' | 'SOFT_DELETE' | 'RESTORE' | 'DELETE';

/**
 * Snapshot de una tarea en un momento dado.
 * Los campos son opcionales porque la forma depende de la versión del
 * esquema cuando se registró. Los tratamos como un objeto libre.
 */
export interface AuditSnapshot {
  readonly id?: string;
  readonly title?: string;
  readonly description?: string;
  readonly priority?: string;
  readonly due_date?: string | null;
  readonly completed?: boolean;
  readonly deleted_at?: string | null;
  readonly created_at?: string;
  readonly updated_at?: string;
  readonly [key: string]: unknown;
}

/**
 * Una entrada del historial de cambios.
 */
export interface AuditEntry {
  readonly id: number;
  readonly task_id: string;
  readonly action: AuditAction;
  readonly old_data?: AuditSnapshot;
  readonly new_data?: AuditSnapshot;
  readonly changed_by: string;
  readonly changed_at: string;
}