import type { Priority } from './task.model';

/**
 * Cuerpo del POST /api/tasks/.
 * No incluye id, created_at, etc. porque el backend los genera.
 */
export interface CreateTaskInput {
  title: string;
  description?: string;
  priority?: Priority;
  due_date?: string | null;
  tag_names?: string[];
}

/**
 * Cuerpo del PUT /api/tasks/{id}.
 *
 * Todos los campos son opcionales porque el endpoint es parcial:
 * solo actualiza lo que se envía. Los campos `undefined` se omiten
 * en el JSON (recuerda: `JSON.stringify({a: undefined})` = `{}`).
 *
 * Distinción importante:
 *   - `title: undefined` → no se envía → se mantiene el valor actual.
 *   - `title: ''`       → se envía vacío → el backend valida y rechaza.
 *   - `completed: undefined` → no se envía.
 *   - `completed: false`     → se envía false explícitamente.
 *
 * Esta es la razón por la que el backend usa punteros en UpdateTaskInput.
 * Aquí replicamos esa semántica con `?`.
 */
export interface UpdateTaskInput {
  title?: string;
  description?: string;
  priority?: Priority;
  due_date?: string | null;
  completed?: boolean;
  tag_names?: string[];
}

/**
 * Filtros de búsqueda del listado.
 * Todos opcionales; solo se envían al backend los que estén definidos.
 */
export interface TaskFilter {
  include_deleted?: boolean;
  completed?: boolean;
  priority?: Priority;
  due_before?: string;
  due_after?: string;
  search?: string;
  tags?: string[];
}