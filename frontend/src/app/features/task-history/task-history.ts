import { Component, inject, input, OnInit, signal } from '@angular/core';
import { RouterLink } from '@angular/router';

import type { AuditEntry } from '@core/models';
import { TaskService, type TaskServiceError } from '@core/services/task.service';
import { auditActionClass, auditActionLabel, formatDateTime } from '@core/utils/format';

/**
 * Página de historial de cambios de una tarea.
 * Solo lectura. Consume el endpoint GET /tasks/{id}/audit.
 */
@Component({
  selector: 'app-task-history',
  imports: [RouterLink],
  templateUrl: './task-history.html',
  styleUrl: './task-history.scss',
})
export class TaskHistory implements OnInit {
  private readonly taskService = inject(TaskService);

  readonly id = input.required<string>();

  protected readonly entries = signal<AuditEntry[]>([]);
  protected readonly loading = signal(false);
  protected readonly error = signal<string | null>(null);

  // Exponemos helpers al template.
  protected readonly auditActionLabel = auditActionLabel;
  protected readonly auditActionClass = auditActionClass;
  protected readonly formatDateTime = formatDateTime;

  ngOnInit(): void {
    this.load();
  }

  private load(): void {
    this.loading.set(true);
    this.error.set(null);

    this.taskService.getAudit(this.id()).subscribe({
      next: (entries) => {
        this.entries.set(entries);
        this.loading.set(false);
      },
      error: (err: TaskServiceError) => {
        this.error.set(err.message);
        this.loading.set(false);
      },
    });
  }

  /**
   * Devuelve la lista de campos que cambiaron entre old_data y new_data.
   * Se usa para mostrar un resumen legible en el template.
   */
  protected changedFields(entry: AuditEntry): string[] {
    if (entry.action !== 'UPDATE') return [];
    const oldData = entry.old_data ?? {};
    const newData = entry.new_data ?? {};

    const ignored = new Set(['updated_at']); // Cambia siempre; no aporta.
    const fields: string[] = [];

    for (const key of Object.keys(newData)) {
      if (ignored.has(key)) continue;
      const before = (oldData as Record<string, unknown>)[key];
      const after = (newData as Record<string, unknown>)[key];
      if (JSON.stringify(before) !== JSON.stringify(after)) {
        fields.push(key);
      }
    }
    return fields;
  }

  /**
   * Formatea un valor de snapshot para mostrar.
   */
  protected formatValue(value: unknown): string {
    if (value === null || value === undefined) return '—';
    if (typeof value === 'boolean') return value ? 'Sí' : 'No';
    if (typeof value === 'string') return value;
    return JSON.stringify(value);
  }
}