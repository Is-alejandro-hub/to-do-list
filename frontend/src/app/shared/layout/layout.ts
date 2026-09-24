import { Component } from '@angular/core';
import { RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';

/**
 * Layout principal de la aplicación.
 *
 * Contiene el header persistente y el router-outlet donde se montan
 * las rutas hijas. Al ser un componente de layout, se mantiene vivo
 * durante toda la sesión y no se destruye al navegar entre rutas.
 */
@Component({
  selector: 'app-layout',
  imports: [RouterOutlet, RouterLink, RouterLinkActive],
  template: `
    <div class="app-shell">
      <header class="app-header">
        <div class="app-header__content">
          <a routerLink="/tasks" class="app-header__brand">
            <span class="app-header__logo">✓</span>
            <span class="app-header__title">To-Do List</span>
          </a>

          <nav class="app-header__nav">
            <a
              routerLink="/tasks"
              routerLinkActive="is-active"
              [routerLinkActiveOptions]="{ exact: true }"
              class="app-header__link"
            >
              Listado
            </a>
            <a
              routerLink="/tasks/new"
              routerLinkActive="is-active"
              class="app-header__link app-header__link--primary"
            >
              + Nueva tarea
            </a>
          </nav>
        </div>
      </header>

      <main class="app-main">
        <router-outlet />
      </main>
    </div>
  `,
  styles: `
    .app-shell {
      min-height: 100vh;
      display: flex;
      flex-direction: column;
      background: #f5f7fa;
    }

    .app-header {
      background: #ffffff;
      border-bottom: 1px solid #e2e8f0;
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
      position: sticky;
      top: 0;
      z-index: 10;
    }

    .app-header__content {
      max-width: 1100px;
      margin: 0 auto;
      padding: 1rem 1.5rem;
      display: flex;
      justify-content: space-between;
      align-items: center;
    }

    .app-header__brand {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      text-decoration: none;
      color: #1a202c;
      font-weight: 700;
      font-size: 1.25rem;
    }

    .app-header__logo {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      width: 32px;
      height: 32px;
      border-radius: 8px;
      background: #3182ce;
      color: white;
      font-size: 1.1rem;
    }

    .app-header__nav {
      display: flex;
      gap: 1rem;
      align-items: center;
    }

    .app-header__link {
      text-decoration: none;
      color: #4a5568;
      font-weight: 500;
      padding: 0.5rem 0.75rem;
      border-radius: 6px;
      transition: background 0.15s, color 0.15s;
    }

    .app-header__link:hover {
      background: #edf2f7;
      color: #1a202c;
    }

    .app-header__link.is-active {
      background: #edf2f7;
      color: #1a202c;
    }

    .app-header__link--primary {
      background: #3182ce;
      color: white;
    }

    .app-header__link--primary:hover {
      background: #2b6cb0;
      color: white;
    }

    .app-header__link--primary.is-active {
      background: #2b6cb0;
      color: white;
    }

    .app-main {
      flex: 1;
      max-width: 1100px;
      width: 100%;
      margin: 0 auto;
      padding: 2rem 1.5rem;
    }
  `,
})
export class Layout {}