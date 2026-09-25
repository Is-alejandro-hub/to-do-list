import { Routes } from '@angular/router';

/**
 * Rutas de la aplicación.
 *
 * Usamos lazy loading (`loadComponent`) para cada ruta de feature.
 * Beneficio: el bundle inicial es pequeño; cada feature se carga
 * solo cuando el usuario navega a ella.
 *
 * `withComponentInputBinding` está activado en app.config.ts, así que
 * los params de ruta (ej: :id) llegan automáticamente como @Input() en
 * los componentes.
 */
export const routes: Routes = [
  {
    path: '',
    redirectTo: 'tasks',
    pathMatch: 'full',
  },
  {
    path: 'tasks',
    loadComponent: () =>
      import('./features/task-list/task-list').then((m) => m.TaskList),
    title: 'Tareas',
  },
  {
    path: 'tasks/new',
    loadComponent: () =>
      import('./features/task-form/task-form').then((m) => m.TaskForm),
    title: 'Nueva tarea',
  },
  {
    path: 'tasks/:id/edit',
    loadComponent: () =>
      import('./features/task-form/task-form').then((m) => m.TaskForm),
    title: 'Editar tarea',
  },
    {
    path: 'tasks/:id/history',
    loadComponent: () =>
      import('./features/task-history/task-history').then((m) => m.TaskHistory),
    title: 'Historial de la tarea',
  },
  {
    path: '**',
    redirectTo: 'tasks',
  },
];