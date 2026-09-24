import { Component, computed, inject, OnInit, signal } from '@angular/core';
import { Router } from '@angular/router';

import type { Task, TaskFilter, Priority } from '@core/models';
import { PRIORITIES } from '@core/models';
import { TaskStore } from '@core/store/task.store';
import { TaskCard } from './task-card/task-card';

/**
 * Página principal del listado de tareas.
 *
 * Responsabilidades:
 *  - Leer el estado del TaskStore (signals).
 *  - Orquestar la carga al inicializar.
 *  - Manejar filtros locales (búsqueda, prioridad, mostrar eliminadas).
 *  - Traducir eventos de las TaskCard en acciones del store.
 *  - Navegar al formulario de edición.
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

  // ─── Estado del listado desde el store ─────────────────────────────
  protected readonly tasks = this.store.tasks;
  protected readonly loading = this.store.loading;
  protected readonly error = this.store.error;
  protected readonly isEmpty = this.store.isEmpty;
  protected readonly totalTasks = this.store.totalTasks;
  protected readonly completedTasks = this.store.completedTasks;
  protected readonly pendingTasks = this.store.pendingTasks;

  // ─── Filtros locales ───────────────────────────────────────────────
  protected readonly search = signal('');
  protected readonly priorityFilter = signal<Priority | null>(null);
  protected readonly showDeleted = signal(false);

  protected readonly priorities = PRIORITIES;

  /**
   * El filtro activo, como objeto para enviar al backend.
   * Es un computed: se recalcula solo cuando cambian los signals.
   */
  protected readonly activeFilter = computed<TaskFilter>(() => {
    const filter: TaskFilter = {};

    const searchValue = this.search().trim();
    if (searchValue) filter.search = searchValue;

    const priority = this.priorityFilter();
    if (priority) filter.priority = priority;

    if (this.showDeleted()) filter.include_deleted = true;

    return filter;
  });

  ngOnInit(): void {
    this.load();
    this.store.loadTags();
  }

  // ─── Acciones desde el template ────────────────────────────────────

  protected load(): void {
    this.store.load(this.activeFilter());
  }

  protected onSearchChange(value: string): void {
    this.search.set(value);
    this.load();
  }

  protected onPriorityChange(value: string): void {
    this.priorityFilter.set(value === '' ? null : (value as Priority));
    this.load();
  }

  protected onShowDeletedChange(checked: boolean): void {
    this.showDeleted.set(checked);
    this.load();
  }

  protected clearFilters(): void {
    this.search.set('');
    this.priorityFilter.set(null);
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
}