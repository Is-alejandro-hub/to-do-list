import { HttpClient, HttpErrorResponse, HttpHeaders, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable, throwError } from 'rxjs';
import { catchError } from 'rxjs/operators';

import type {
  CreateTaskInput,
  UpdateTaskInput,
  TaskFilter,
  Task,
  Tag,
  ApiError,
  ValidationErrorDetail,
} from '@core/models';
import { isApiError } from '@core/models';
import { generateUUID } from '@core/utils/uuid';

/**
 * Error tipado que envuelve cualquier fallo de la API.
 * Es lo que los componentes verán cuando algo falle.
 */
export interface TaskServiceError {
  readonly status: number;
  readonly message: string;
  readonly details: readonly ValidationErrorDetail[];
}

/**
 * Servicio HTTP único para tareas y etiquetas.
 *
 * Responsabilidades:
 *  - Construir URLs relativas (/api/...) que pasan por el proxy.
 *  - Añadir la cabecera Idempotency-Key en POST.
 *  - Normalizar errores del backend a TaskServiceError.
 *  - Serializar filtros a query params.
 *
 * NO tiene estado propio. Los componentes manejan su propio estado
 * con signals. Este servicio es puro I/O.
 */
@Injectable({ providedIn: 'root' })
export class TaskService {
  private readonly http = inject(HttpClient);

  private readonly baseUrl = '/api/tasks';
  private readonly tagsUrl = '/api/tags';

  // ==========================================================================
  // LECTURA
  // ==========================================================================

  /**
   * Lista tareas con filtros opcionales.
   */
  list(filter?: TaskFilter): Observable<Task[]> {
    const params = this.buildFilterParams(filter);
    return this.http
      .get<Task[]>(`${this.baseUrl}/`, { params })
      .pipe(catchError(this.handleError));
  }

  /**
   * Obtiene una tarea por ID.
   */
  getById(id: string, includeDeleted = false): Observable<Task> {
    const params = includeDeleted
      ? new HttpParams().set('include_deleted', 'true')
      : undefined;

    return this.http
      .get<Task>(`${this.baseUrl}/${id}`, { params })
      .pipe(catchError(this.handleError));
  }

  /**
   * Lista el catálogo completo de etiquetas.
   */
  listTags(): Observable<Tag[]> {
    return this.http
      .get<Tag[]>(this.tagsUrl)
      .pipe(catchError(this.handleError));
  }

  // ==========================================================================
  // ESCRITURA
  // ==========================================================================

  /**
   * Crea una tarea.
   *
   * Genera un UUID nuevo como Idempotency-Key. Si el navegador reintenta
   * la misma petición por un fallo de red, el backend detectará el
   * reintento y no creará duplicados.
   *
   * El UUID se genera DENTRO de este método, no se recibe como argumento.
   * Esto garantiza que cada llamada a `create()` sea una operación lógica
   * nueva, con su propia clave.
   */
  create(input: CreateTaskInput): Observable<Task> {
    const headers = new HttpHeaders({
      'Idempotency-Key': generateUUID(),
    });

    return this.http
      .post<Task>(`${this.baseUrl}/`, input, { headers })
      .pipe(catchError(this.handleError));
  }

  /**
   * Actualiza una tarea.
   *
   * Nota: solo se envían los campos presentes en `input`. Los campos
   * `undefined` se omiten automáticamente por JSON.stringify.
   */
  update(id: string, input: UpdateTaskInput): Observable<Task> {
    return this.http
      .put<Task>(`${this.baseUrl}/${id}`, input)
      .pipe(catchError(this.handleError));
  }

  /**
   * Marca una tarea como eliminada (soft delete).
   * Devuelve Observable<void> porque el backend responde 204 sin body.
   */
  softDelete(id: string): Observable<void> {
    return this.http
      .delete<void>(`${this.baseUrl}/${id}`)
      .pipe(catchError(this.handleError));
  }

  /**
   * Restaura una tarea previamente eliminada.
   */
  restore(id: string): Observable<void> {
    return this.http
      .post<void>(`${this.baseUrl}/${id}/restore`, null)
      .pipe(catchError(this.handleError));
  }

  // ==========================================================================
  // HELPERS INTERNOS
  // ==========================================================================

  /**
   * Convierte un TaskFilter en HttpParams.
   *
   * Reglas:
   *  - Los valores undefined se omiten.
   *  - Los booleanos se envían como "true"/"false".
   *  - Los arrays se envían repetidos (?tags=a&tags=b).
   */
  private buildFilterParams(filter?: TaskFilter): HttpParams {
    let params = new HttpParams();
    if (!filter) return params;

    if (filter.include_deleted !== undefined) {
      params = params.set('include_deleted', String(filter.include_deleted));
    }
    if (filter.completed !== undefined) {
      params = params.set('completed', String(filter.completed));
    }
    if (filter.priority !== undefined) {
      params = params.set('priority', filter.priority);
    }
    if (filter.due_before !== undefined) {
      params = params.set('due_before', filter.due_before);
    }
    if (filter.due_after !== undefined) {
      params = params.set('due_after', filter.due_after);
    }
    if (filter.search !== undefined && filter.search.trim() !== '') {
      params = params.set('search', filter.search.trim());
    }
    if (filter.tags !== undefined && filter.tags.length > 0) {
      for (const tag of filter.tags) {
        params = params.append('tags', tag);
      }
    }

    return params;
  }

  /**
   * Normaliza cualquier error HTTP a un TaskServiceError tipado.
   *
   * Se pasa como arrow function al `catchError` para preservar el `this`.
   * Si fuera un método tradicional, `this` sería undefined.
   */
  private handleError = (error: HttpErrorResponse): Observable<never> => {
    const normalized = this.normalizeError(error);
    return throwError(() => normalized);
  };

  private normalizeError(error: HttpErrorResponse): TaskServiceError {
    // Errores de red o cliente: sin status del servidor.
    if (error.status === 0) {
      return {
        status: 0,
        message: 'No se pudo conectar con el servidor. Verifica tu conexión.',
        details: [],
      };
    }

    // Errores con cuerpo de la API.
    if (isApiError(error.error)) {
      return {
        status: error.status,
        message: error.error.error,
        details: error.error.details ?? [],
      };
    }

    // Errores genéricos.
    return {
      status: error.status,
      message: error.statusText || 'Error desconocido del servidor',
      details: [],
    };
  }
}