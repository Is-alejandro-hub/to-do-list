/**
 * Prioridad de una tarea.
 *
 * Usamos union type en lugar de enum de TypeScript porque:
 *  - El JSON del backend envía strings ("LOW", "MEDIUM", "HIGH").
 *  - Los union types son más ligeros (no generan código en runtime).
 *  - Con `const` funcionan igual que un enum para autocompletado.
 */
export type Priority = 'LOW' | 'MEDIUM' | 'HIGH';

/**
 * Constante con los valores válidos de Priority.
 * Útil para iterar en selects y para validaciones.
 */
export const PRIORITIES: readonly Priority[] = ['LOW', 'MEDIUM', 'HIGH'] as const;

/**
 * Etiqueta reutilizable.
 * Refleja el modelo del backend en internal/domain/task.go.
 */
export interface Tag {
  readonly id: string;
  readonly name: string;
  readonly created_at: string; // ISO 8601
}

/**
 * Tarea completa, tal como la devuelve el backend.
 *
 * Nota sobre las fechas: el backend envía strings ISO 8601 (`2026-01-15T15:00:00Z`).
 * NO las parseamos a Date en el modelo porque:
 *   - La conversión Date → string al enviar al backend es trivial.
 *   - Mantener el tipo string evita errores sutiles de zona horaria.
 *   - El pipe `date` de Angular convierte al mostrar.
 */
export interface Task {
  readonly id: string;
  readonly title: string;
  readonly description: string;
  readonly priority: Priority;
  readonly due_date: string | null;
  readonly completed: boolean;
  readonly deleted_at: string | null;
  readonly created_at: string;
  readonly updated_at: string;
  readonly tags?: readonly Tag[];
}

/**
 * Helper de tipo: una tarea que sabemos que está eliminada.
 * Útil para tipar listados mixtos y evitar comprobaciones redundantes.
 */
export type DeletedTask = Task & { readonly deleted_at: string };