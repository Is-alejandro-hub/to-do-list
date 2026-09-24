import { Component, input } from '@angular/core';

/**
 * Formulario de crear/editar tarea. Placeholder por ahora; se implementa en 4.8.
 *
 * `input()` es la API moderna de Angular 21 para recibir datos.
 * El param de ruta `:id` llega automáticamente gracias a
 * withComponentInputBinding() en app.config.ts.
 */
@Component({
  selector: 'app-task-form',
  template: `
    <div class="placeholder">
      <h1>{{ id() ? 'Editar tarea' : 'Nueva tarea' }}</h1>
      <p>Se implementará en el paso 4.8.</p>
      @if (id()) {
        <p><small>ID: {{ id() }}</small></p>
      }
    </div>
  `,
  styles: `
    .placeholder {
      padding: 2rem;
      background: white;
      border-radius: 12px;
      text-align: center;
      color: #4a5568;
    }
  `,
})
export class TaskForm {
  /**
   * Viene del param de ruta `:id`.
   * - undefined en modo crear (/tasks/new)
   * - string en modo editar (/tasks/:id/edit)
   */
  readonly id = input<string>();
}