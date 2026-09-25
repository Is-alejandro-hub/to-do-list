import { Component, computed, effect, inject, OnInit, signal } from '@angular/core';
import { Router } from '@angular/router';

import type { Task, TaskFilter, Priority } from '@core/models';
import { PRIORITIES } from '@core/models';
import { TaskStore } from '@core/store/task.store';
import { TaskCard } from './task-card/task-card';

/**
 * Página principal del listado de tareas.
 *
 * Gestión de filtros:
 *  - Búsqueda con debounce de 300ms.
 *  - Prioridad, rango de fechas y etiquetas.
 *  - Indicador de filtros activos con botón de limpieza.
 */
@Component({
  selector: 'app-task-list',
  imports: [TaskCard],
  templateUrl: './task-list.html',
  styleUrl: './task-list.scss',
})
export class TaskList implements OnInit {
  private readonly store = inject(TaskStore);
  private readonly router = inject(Router);

  // ─── Estado del listado ────────────────────────────────────────────
  protected readonly tasks = this.store.tasks;
  protected readonly loading = this.store.loading;
  protected readonly error = this.store.error;
  protected readonly isEmpty = this.store.isEmpty;
  protected readonly totalTasks = this.store.totalTasks;
  protected readonly completedTasks = this.store.completedTasks;
  protected readonly pendingTasks = this.store.pendingTasks;
  protected readonly availableTags = this.store.tags;

  // ─── Filtros ───────────────────────────────────────────────────────
  // searchInput: lo que el usuario escribe (sin debounce).
  // search: lo que llega al backend (con debounce).
  protected readonly searchInput = signal('');
  protected readonly search = signal('');
  protected readonly priorityFilter = signal<Priority | null>(null);
  protected readonly dueAfter = signal<string>('');
  protected readonly dueBefore = signal<string>('');
  protected readonly selectedTags = signal<string[]>([]);
  protected readonly showDeleted = signal(false);

  protected readonly priorities = PRIORITIES;

  /**
   * Filtro efectivo a enviar al backend. Es un computed: solo se
   * recalcula cuando cambia alguno de los signals que lee.
   */
  protected readonly activeFilter = computed<TaskFilter>(() => {
    const filter: TaskFilter = {};

    const s = this.search().trim();
    if (s) filter.search = s;

    const p = this.priorityFilter();
    if (p) filter.priority = p;

    const after = this.dueAfter();
    if (after) filter.due_after = this.dateInputToIsoStart(after);

    const before = this.dueBefore();
    if (before) filter.due_before = this.dateInputToIsoEnd(before);

    const tags = this.selectedTags();
    if (tags.length > 0) filter.tags = tags;

    if (this.showDeleted()) filter.include_deleted = true;

    return filter;
  });

  /**
   * Número total de filtros activos (excluye "mostrar eliminadas").
   * Se usa para el indicador visual.
   */
  protected readonly activeFiltersCount = computed(() => {
    let count = 0;
    if (this.searchInput().trim()) count++;
    if (this.priorityFilter()) count++;
    if (this.dueAfter()) count++;
    if (this.dueBefore()) count++;
    if (this.selectedTags().length > 0) count++;
    return count;
  });

  protected readonly hasActiveFilters = computed(() => this.activeFiltersCount() > 0);

  /**
   * Indica si el listado está vacío POR FILTROS (no porque no haya tareas).
   * Útil para mostrar un mensaje distinto en el empty state.
   */
  protected readonly emptyByFilters = computed(
    () => !this.loading() && this.tasks().length === 0 && this.hasActiveFilters()
  );

  /**
   * Indica si el listado está vacío porque NO HAY tareas en absoluto.
   */
  protected readonly emptyByNoTasks = computed(
    () => !this.loading() && this.tasks().length === 0 && !this.hasActiveFilters()
  );

  // ─── Debounce de la búsqueda ───────────────────────────────────────
  constructor() {
  let isFirstRun = true;
  effect((onCleanup) => {
    const value = this.searchInput();

    // Salta la primera ejecución para no duplicar la carga inicial
    // que ya hace ngOnInit.
    if (isFirstRun) {
      isFirstRun = false;
      return;
    }

    const timer = setTimeout(() => {
      this.search.set(value);
      this.load();
    }, 300);
    onCleanup(() => clearTimeout(timer));
  });
}

  ngOnInit(): void {
    this.load();
    this.store.loadTags();
  }

  // ─── Acciones ──────────────────────────────────────────────────────

  protected load(): void {
    this.store.load(this.activeFilter());
  }

  protected onSearchChange(value: string): void {
    this.searchInput.set(value);
  }

  protected onPriorityChange(value: string): void {
    this.priorityFilter.set(value === '' ? null : (value as Priority));
    this.load();
  }

  protected onDueAfterChange(value: string): void {
    this.dueAfter.set(value);
    this.load();
  }

  protected onDueBeforeChange(value: string): void {
    this.dueBefore.set(value);
    this.load();
  }

  protected onShowDeletedChange(checked: boolean): void {
    this.showDeleted.set(checked);
    this.load();
  }

  /**
   * Alterna la selección de un tag en el filtro.
   */
  protected toggleTag(tagName: string): void {
    const current = this.selectedTags();
    const next = current.includes(tagName)
      ? current.filter((t) => t !== tagName)
      : [...current, tagName];
    this.selectedTags.set(next);
    this.load();
  }

  protected isTagSelected(tagName: string): boolean {
    return this.selectedTags().includes(tagName);
  }

  protected clearFilters(): void {
    this.searchInput.set('');
    this.search.set('');
    this.priorityFilter.set(null);
    this.dueAfter.set('');
    this.dueBefore.set('');
    this.selectedTags.set([]);
    this.showDeleted.set(false);
    this.load();
  }

  protected onToggleCompleted(task: Task): void {
    this.store.update(task.id, { completed: !task.completed }, this.activeFilter());
  }

  protected onEdit(task: Task): void {
    this.router.navigate(['/tasks', task.id, 'edit']);
  }

  protected onRemove(task: Task): void {
    if (!confirm(`¿Eliminar la tarea "${task.title}"?`)) return;
    this.store.softDelete(task.id, this.activeFilter());
  }

  protected onRestore(task: Task): void {
    this.store.restore(task.id, this.activeFilter());
  }

  protected onClearError(): void {
    this.store.clearError();
  }

  protected onNewTask(): void {
    this.router.navigate(['/tasks', 'new']);
  }

  // ─── Helpers de fecha ──────────────────────────────────────────────

  /**
   * Convierte "YYYY-MM-DD" al ISO 8601 con hora 00:00:00 UTC.
   * Se usa para `due_after` (todo lo que venza a partir de esa fecha).
   */
  private dateInputToIsoStart(dateInput: string): string {
    return new Date(`${dateInput}T00:00:00.000Z`).toISOString();
  }

  /**
   * Convierte "YYYY-MM-DD" al ISO 8601 con hora 23:59:59 UTC.
   * Se usa para `due_before` (todo lo que venza hasta el final de ese día).
   */
  private dateInputToIsoEnd(dateInput: string): string {
    return new Date(`${dateInput}T23:59:59.999Z`).toISOString();
  }
}