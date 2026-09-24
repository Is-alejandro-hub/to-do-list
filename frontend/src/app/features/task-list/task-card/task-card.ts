import { Component, input, output } from '@angular/core';

import type { Task } from '@core/models';
import {
  priorityLabel,
  priorityClass,
  formatDateTime,
  formatDate,
  isPast,
} from '@core/utils/format';

/**
 * Tarjeta de una tarea. Componente "tonto" (presentacional):
 *  - Recibe la tarea por input.
 *  - Emite eventos por output para que el padre decida qué hacer.
 *
 * No conoce el store ni el servicio HTTP. Esta separación permite
 * testearlo fácilmente y reusarlo en otros contextos.
 */
@Component({
  selector: 'app-task-card',
  templateUrl: './task-card.html',
  styleUrl: './task-card.scss',
})
export class TaskCard {
  readonly task = input.required<Task>();

  readonly toggleCompleted = output<Task>();
  readonly edit = output<Task>();
  readonly remove = output<Task>();
  readonly restore = output<Task>();

  // Helpers expuestos al template.
  protected readonly priorityLabel = priorityLabel;
  protected readonly priorityClass = priorityClass;
  protected readonly formatDate = formatDate;
  protected readonly formatDateTime = formatDateTime;
  protected readonly isPast = isPast;

  protected onToggle(): void {
    this.toggleCompleted.emit(this.task());
  }

  protected onEdit(): void {
    this.edit.emit(this.task());
  }

  protected onRemove(): void {
    this.remove.emit(this.task());
  }

  protected onRestore(): void {
    this.restore.emit(this.task());
  }
}