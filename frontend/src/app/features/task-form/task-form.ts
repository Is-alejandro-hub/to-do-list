import { Component, computed, effect, inject, input, OnInit, signal } from '@angular/core';
import { FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';

import type { CreateTaskInput, Priority, Task, UpdateTaskInput } from '@core/models';
import { PRIORITIES } from '@core/models';
import { TaskStore } from '@core/store/task.store';
import { TaskService } from '@core/services/task.service';

/**
 * Formulario de crear y editar tarea.
 *
 * El modo se decide automáticamente:
 *  - input() `id` viene undefined → modo crear
 *  - input() `id` viene con valor → modo editar (carga la tarea)
 *
 * Al guardar:
 *  - Modo crear: llama a store.create → POST → idempotencia automática
 *  - Modo editar: llama a store.update → PUT
 *  - Ambos casos: navega de vuelta a /tasks
 */
@Component({
  selector: 'app-task-form',
  imports: [ReactiveFormsModule, RouterLink],
  templateUrl: './task-form.html',
  styleUrls: ['./task-form.scss'],
})
export class TaskForm implements OnInit {
  private readonly store = inject(TaskStore);
  private readonly taskService = inject(TaskService);
  private readonly router = inject(Router);

  // ─── Input de ruta (viene del param :id) ───────────────────────────
  readonly id = input<string>();

  // ─── Modo derivado ─────────────────────────────────────────────────
  protected readonly isEditMode = computed(() => !!this.id());

  // ─── Estado local ──────────────────────────────────────────────────
  protected readonly loadingTask = signal(false);
  protected readonly saving = signal(false);
  protected readonly serverError = signal<string | null>(null);
  protected readonly priorities = PRIORITIES;

  /**
   * Formulario tipado.
   *
   * `nonNullable: true` reduce el tipo de `string | null` a `string`,
   * eliminando comprobaciones defensivas en el template.
   */
  protected readonly form = new FormGroup({
    title: new FormControl<string>('', {
      nonNullable: true,
      validators: [
        Validators.required,
        Validators.maxLength(255),
      ],
    }),
    description: new FormControl<string>('', { nonNullable: true }),
    priority: new FormControl<Priority>('MEDIUM', { nonNullable: true }),
    dueDate: new FormControl<string>('', { nonNullable: true }),
    completed: new FormControl<boolean>(false, { nonNullable: true }),
    tagNames: new FormControl<string>('', { nonNullable: true }),
  });

  // ─── Ciclo de vida ─────────────────────────────────────────────────

  ngOnInit(): void {
    if (this.isEditMode()) {
      this.loadTask();
    }
  }

  // ─── Carga en modo editar ──────────────────────────────────────────

  private loadTask(): void {
    const taskId = this.id();
    if (!taskId) return;

    this.loadingTask.set(true);
    this.serverError.set(null);

    this.taskService.getById(taskId).subscribe({
      next: (task) => {
        this.populateForm(task);
        this.loadingTask.set(false);
      },
      error: (err) => {
        this.serverError.set(err.message);
        this.loadingTask.set(false);
      },
    });
  }

  /**
   * Vuelca una Task del backend al formulario.
   *
   * El backend envía la fecha en ISO 8601 UTC (2026-01-15T15:00:00Z).
   * El input type="date" espera YYYY-MM-DD. Hay que convertir.
   */
  private populateForm(task: Task): void {
    this.form.patchValue({
      title: task.title,
      description: task.description,
      priority: task.priority,
      dueDate: task.due_date ? this.isoToDateInput(task.due_date) : '',
      completed: task.completed,
      tagNames: (task.tags ?? []).map((t) => t.name).join(', '),
    });
  }

  // ─── Envío ─────────────────────────────────────────────────────────

  protected onSubmit(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    this.saving.set(true);
    this.serverError.set(null);

    const formValue = this.form.getRawValue();

    if (this.isEditMode()) {
      this.updateExisting(formValue);
    } else {
      this.createNew(formValue);
    }
  }

  private createNew(formValue: ReturnType<typeof this.form.getRawValue>): void {
    const input: CreateTaskInput = {
      title: formValue.title.trim(),
      description: formValue.description.trim() || undefined,
      priority: formValue.priority,
      due_date: formValue.dueDate ? this.dateInputToIso(formValue.dueDate) : null,
      tag_names: this.parseTags(formValue.tagNames),
    };

    this.taskService.create(input).subscribe({
      next: () => {
        this.saving.set(false);
        // Refrescamos el listado sin filtros para que se vea la nueva tarea.
        this.store.load();
        this.router.navigate(['/tasks']);
      },
      error: (err) => {
        this.serverError.set(err.message);
        this.saving.set(false);
      },
    });
  }

  private updateExisting(formValue: ReturnType<typeof this.form.getRawValue>): void {
    const taskId = this.id();
    if (!taskId) return;

    const input: UpdateTaskInput = {
      title: formValue.title.trim(),
      description: formValue.description.trim(),
      priority: formValue.priority,
      due_date: formValue.dueDate ? this.dateInputToIso(formValue.dueDate) : null,
      completed: formValue.completed,
      tag_names: this.parseTags(formValue.tagNames),
    };

    this.taskService.update(taskId, input).subscribe({
      next: () => {
        this.saving.set(false);
        this.store.load();
        this.router.navigate(['/tasks']);
      },
      error: (err) => {
        this.serverError.set(err.message);
        this.saving.set(false);
      },
    });
  }

  // ─── Helpers ───────────────────────────────────────────────────────

  /**
   * Convierte el string de tags ("a, b, c") a un array limpio.
   */
  private parseTags(raw: string): string[] {
    return raw
      .split(',')
      .map((t) => t.trim())
      .filter((t) => t.length > 0);
  }

  /**
   * Convierte ISO 8601 ("2026-01-15T15:00:00Z") a "2026-01-15"
   * para el input type="date".
   */
  private isoToDateInput(iso: string): string {
    const d = new Date(iso);
    const year = d.getFullYear();
    const month = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  }

  /**
   * Convierte "2026-01-15" del input date a ISO 8601 UTC.
   * Usamos 23:59:59 del día para que la fecha de vencimiento se
   * interprete como "vence al final del día".
   */
  private dateInputToIso(dateInput: string): string {
    return new Date(`${dateInput}T23:59:59.000Z`).toISOString();
  }

  // ─── Acceso al template ────────────────────────────────────────────

  protected get title() { return this.form.controls.title; }
  protected get description() { return this.form.controls.description; }
  protected get priority() { return this.form.controls.priority; }
  protected get dueDate() { return this.form.controls.dueDate; }
  protected get completed() { return this.form.controls.completed; }
  protected get tagNames() { return this.form.controls.tagNames; }
}