import { DestroyRef, Injectable, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';

import type {
  Task,
  Tag,
  CreateTaskInput,
  UpdateTaskInput,
  TaskFilter,
} from '@core/models';
import { TaskService, type TaskServiceError } from '@core/services/task.service';

/**
 * Store central de tareas y etiquetas.
 *
 * Responsabilidades:
 *  - Mantener la fuente única de verdad del estado de tareas y etiquetas.
 *  - Ejecutar operaciones de negocio llamando al TaskService.
 *  - Notificar a los componentes vía signals cuando el estado cambia.
 *
 * Patrón:
 *  - Estado privado con signals escribibles (#tasks, #loading, #error).
 *  - Selectores públicos con signals de solo lectura.
 *  - Métodos que mutan el estado y que los componentes llaman.
 *
 * Los componentes NUNCA escriben al estado directamente. Solo leen los
 * selectores y llaman a los métodos. Es lo que hace el estado predecible.
 */
@Injectable({ providedIn: 'root' })
export class TaskStore {
  private readonly taskService = inject(TaskService);
  private readonly destroyRef = inject(DestroyRef);

  // ==========================================================================
  // ESTADO PRIVADO (escribible)
  // ==========================================================================
  // El prefijo # hace que estos campos sean verdaderamente privados en JS,
  // no solo "por convención" como el guion bajo. Nadie fuera de la clase
  // puede leerlos ni mutearlos directamente.

  readonly #tasks = signal<Task[]>([]);
  readonly #tags = signal<Tag[]>([]);
  readonly #loading = signal(false);
  readonly #saving = signal(false);
  readonly #error = signal<TaskServiceError | null>(null);

  // ==========================================================================
  // SELECTORES PÚBLICOS (solo lectura)
  // ==========================================================================
  // asReadonly() devuelve un Signal que no tiene métodos de escritura.
  // Los componentes lo pueden leer pero no lo pueden mutar.

  readonly tasks = this.#tasks.asReadonly();
  readonly tags = this.#tags.asReadonly();
  readonly loading = this.#loading.asReadonly();
  readonly saving = this.#saving.asReadonly();
  readonly error = this.#error.asReadonly();

  // ==========================================================================
  // SELECTORES COMPUTADOS (derivados, se recalculan solos)
  // ==========================================================================

  readonly totalTasks = computed(() => this.#tasks().length);
  readonly completedTasks = computed(() =>
    this.#tasks().filter((t) => t.completed).length
  );
  readonly pendingTasks = computed(
    () => this.totalTasks() - this.completedTasks()
  );
  readonly hasError = computed(() => this.#error() !== null);
  readonly isEmpty = computed(
    () => !this.#loading() && this.#tasks().length === 0
  );

  // ==========================================================================
  // ACCIONES
  // ==========================================================================

  /**
   * Carga el listado de tareas con filtros opcionales.
   *
   * Flujo:
   *  1. Marca loading=true y limpia error.
   *  2. Llama al servicio.
   *  3. Al recibir respuesta: actualiza #tasks y loading=false.
   *  4. Al recibir error: guarda el error y loading=false.
   */
  load(filter?: TaskFilter): void {
    this.#loading.set(true);
    this.#error.set(null);

    this.taskService
      .list(filter)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (tasks) => {
          this.#tasks.set(tasks);
          this.#loading.set(false);
        },
        error: (err: TaskServiceError) => {
          this.#error.set(err);
          this.#loading.set(false);
        },
      });
  }

  /**
   * Carga el catálogo de etiquetas. Suele llamarse una sola vez.
   */
  loadTags(): void {
    this.taskService
      .listTags()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (tags) => this.#tags.set(tags),
        error: (err: TaskServiceError) => this.#error.set(err),
      });
  }

  /**
   * Crea una tarea y refresca el listado.
   *
   * Estrategia: en lugar de insertar la nueva tarea en el array local,
   * recargamos el listado completo. Es menos eficiente pero garantiza
   * que el orden y los filtros del backend se respeten.
   *
   * Alternativa (más rápida pero más compleja): insertar en la posición
   * correcta según el ordenamiento actual. No vale la pena para esta app.
   */
  create(input: CreateTaskInput, filter?: TaskFilter): void {
    this.#saving.set(true);
    this.#error.set(null);

    this.taskService
      .create(input)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: () => {
          this.#saving.set(false);
          this.load(filter);
        },
        error: (err: TaskServiceError) => {
          this.#error.set(err);
          this.#saving.set(false);
        },
      });
  }

  /**
   * Actualiza una tarea y refresca el listado.
   */
  update(id: string, input: UpdateTaskInput, filter?: TaskFilter): void {
    this.#saving.set(true);
    this.#error.set(null);

    this.taskService
      .update(id, input)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: () => {
          this.#saving.set(false);
          this.load(filter);
        },
        error: (err: TaskServiceError) => {
          this.#error.set(err);
          this.#saving.set(false);
        },
      });
  }

  /**
   * Marca una tarea como eliminada y refresca el listado.
   */
  softDelete(id: string, filter?: TaskFilter): void {
    this.#saving.set(true);
    this.#error.set(null);

    this.taskService
      .softDelete(id)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: () => {
          this.#saving.set(false);
          this.load(filter);
        },
        error: (err: TaskServiceError) => {
          this.#error.set(err);
          this.#saving.set(false);
        },
      });
  }

  /**
   * Restaura una tarea eliminada y refresca el listado.
   */
  restore(id: string, filter?: TaskFilter): void {
    this.#saving.set(true);
    this.#error.set(null);

    this.taskService
      .restore(id)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: () => {
          this.#saving.set(false);
          this.load(filter);
        },
        error: (err: TaskServiceError) => {
          this.#error.set(err);
          this.#saving.set(false);
        },
      });
  }

  /**
   * Limpia el error actual. Útil después de mostrarlo al usuario.
   */
  clearError(): void {
    this.#error.set(null);
  }
}