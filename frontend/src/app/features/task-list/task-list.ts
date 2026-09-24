import { Component } from '@angular/core';

/**
 * Listado de tareas. Placeholder por ahora; se implementa en 4.7.
 */
@Component({
  selector: 'app-task-list',
  template: `
    <div class="placeholder">
      <h1>Listado de tareas</h1>
      <p>Se implementará en el paso 4.7.</p>
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
export class TaskList {}