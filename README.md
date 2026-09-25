# To-Do List Full Stack

Aplicación full-stack de gestión de tareas con **backend en Go**, **frontend en Angular 21** y **base de datos PostgreSQL**.

Desarrollada como prueba técnica full-stack. Incluye CRUD completo, idempotencia de creación, soft delete, auditoría automática por trigger, etiquetas, prioridades, fechas de vencimiento, filtros avanzados y visualización de historial de cambios.

---

## 📋 Tabla de contenidos

- [Stack tecnológico](#-stack-tecnológico)
- [Características](#-características)
- [Arquitectura](#-arquitectura)
- [Estructura del proyecto](#-estructura-del-proyecto)
- [Requisitos previos](#-requisitos-previos)
- [Instalación y arranque](#-instalación-y-arranque)
- [Variables de entorno](#-variables-de-entorno)
- [API REST](#-api-rest)
- [Base de datos](#-base-de-datos)
- [Funcionalidades adicionales](#-funcionalidades-adicionales)
- [Decisiones de diseño](#-decisiones-de-diseño)
- [Tests](#-tests)
- [Build de producción](#-build-de-producción)
- [Flujo de trabajo con Git](#-flujo-de-trabajo-con-git)
- [Documentación adicional](#-documentación-adicional)

---

## 🛠 Stack tecnológico

| Capa | Tecnología | Versión |
| :--- | :--- | :--- |
| **Frontend** | Angular | 21 (standalone + signals + zoneless) |
| **Backend** | Go | 1.25 |
| **Base de datos** | PostgreSQL | 13+ |
| **Router HTTP** | Chi | v5 |
| **Driver PostgreSQL** | pgx | v5 |
| **Gestión de configuración** | godotenv | v1 |
| **Estilos** | SCSS con design tokens | — |

---

## ✨ Características

### Funcionales (core)

- ✅ **CRUD completo** de tareas (crear, listar, obtener, actualizar, eliminar)
- ✅ **Filtros avanzados**: búsqueda por texto, prioridad, rango de fechas y etiquetas
- ✅ **Estados visuales**: loading, vacío, error, listado
- ✅ **Responsive**: adaptable a móvil, tablet y escritorio
- ✅ **Accesibilidad**: focus rings, ARIA labels, prefers-reduced-motion

### Valor agregado

**Feature 1: Gestión avanzada de tareas**

- **Etiquetas (tags)**: relación muchos-a-muchos, reutilizables entre tareas.
- **Fechas de vencimiento y filtros**: cada tarea puede tener `due_date`; el listado soporta filtros por rango de fechas, prioridad, etiquetas y búsqueda textual con debounce.
- **Prioridad**: enum nativo de PostgreSQL (`LOW`, `MEDIUM`, `HIGH`) con ordenamiento automático.

**Feature 2: Integridad y trazabilidad de datos**

- **Soft delete**: las tareas eliminadas se marcan con `deleted_at` y se pueden restaurar.
- **Audit log**: cada cambio sobre una tarea queda registrado automáticamente por un **trigger de PostgreSQL**, visible desde la UI con línea de tiempo y diff de campos.

**Bonus: Idempotencia de creación**

- El endpoint `POST /tasks/` es idempotente mediante la cabecera `Idempotency-Key`.
- Reintentos por fallos de red no crean duplicados.

---

## 🏗 Arquitectura

### Backend (Go)

Arquitectura en capas inspirada en Clean Architecture:

```
┌─────────────────────────────────────────────────────────┐
│  Handler    → traduce HTTP ↔ dominio                    │
│  Service    → reglas de negocio                         │
│  Repository → acceso a datos (único lugar con SQL)      │
│  Domain     → entidades y errores de negocio            │
└─────────────────────────┬───────────────────────────────┘
                          │ pgx
┌─────────────────────────▼───────────────────────────────┐
│                     PostgreSQL                          │
│  - tasks          - tags          - task_tags           │
│  - task_audit_log - idempotency_keys                    │
└─────────────────────────────────────────────────────────┘
```

**Principio clave**: cada capa solo conoce a la inmediatamente inferior. El dominio no sabe que existe HTTP ni SQL. El repositorio es el único lugar con sentencias SQL.

### Frontend (Angular 21)

Arquitectura reactiva moderna con Signals:

```
┌─────────────────────────────────────────────────────────┐
│  Components (TaskList, TaskForm, TaskHistory, TaskCard) │
│      ↓ lee signals           ↑ llama métodos             │
│  TaskStore (estado reactivo)                            │
│      ↓ llama métodos                                     │
│  TaskService (HTTP puro)                                │
│      ↓                                                   │
│  Backend API                                             │
└─────────────────────────────────────────────────────────┘
```

**Pilares**:
- **Standalone components**: sin `NgModules`, cada componente importa lo que usa.
- **Signals**: estado reactivo nativo, sin RxJS donde no es necesario.
- **Zoneless change detection**: sin `zone.js`, detección de cambios por signals.
- **Control flow moderno**: `@if`, `@for`, `@switch` en lugar de `*ngIf`, `*ngFor`.

---

## 📁 Estructura del proyecto

```
todo-list/
├── backend/
│   ├── cmd/api/                    # Punto de entrada
│   ├── internal/
│   │   ├── config/                 # Carga de variables de entorno
│   │   ├── database/               # Pool de conexiones a PostgreSQL
│   │   ├── domain/                 # Entidades y errores de negocio
│   │   │   ├── audit.go
│   │   │   ├── errors.go
│   │   │   ├── idempotency.go
│   │   │   ├── repository.go       # Interfaces de repositorio
│   │   │   └── task.go
│   │   ├── handler/                # Endpoints HTTP
│   │   │   ├── cors.go
│   │   │   ├── middleware.go
│   │   │   ├── parsing.go
│   │   │   ├── response.go
│   │   │   └── task_handler.go
│   │   ├── repository/             # Acceso a datos (SQL)
│   │   │   ├── audit_repository.go
│   │   │   ├── idempotency_repository.go
│   │   │   └── task_repository.go
│   │   ├── router/                 # Configuración de rutas
│   │   ├── scheduler/              # Worker de limpieza
│   │   └── service/                # Lógica de negocio
│   ├── .env.example
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── app/
│   │   │   ├── core/
│   │   │   │   ├── models/         # Tipos TypeScript del dominio
│   │   │   │   ├── services/       # TaskService (HTTP)
│   │   │   │   ├── store/          # TaskStore (signals)
│   │   │   │   └── utils/          # Helpers (format, uuid)
│   │   │   ├── features/
│   │   │   │   ├── task-form/      # Crear/editar
│   │   │   │   ├── task-history/   # Historial de auditoría
│   │   │   │   └── task-list/      # Listado con filtros
│   │   │   ├── shared/layout/      # Header y shell
│   │   │   ├── app.config.ts       # Providers globales
│   │   │   ├── app.routes.ts       # Rutas con lazy loading
│   │   │   └── app.ts              # Componente raíz
│   │   ├── styles/
│   │   │   ├── _variables.scss     # Design tokens
│   │   │   ├── _mixins.scss        # Mixins reutilizables
│   │   │   └── _buttons.scss       # Estilos de botones
│   │   └── styles.scss             # Estilos globales
│   ├── angular.json
│   ├── proxy.conf.json
│   └── package.json
├── database/
│   └── 01_schema.sql               # Esquema completo
├── docs/
│   ├── decisiones.md               # 20 ADRs documentados
│   └── diagramas.md                # Entidad-relación
└── README.md
```

---

## ✅ Requisitos previos

- **Go 1.25** o superior
- **PostgreSQL 13** o superior
- **Node.js 22 LTS** o superior
- **Angular CLI 21** (`npm install -g @angular/cli@21`)
- **Git**

---

## 🚀 Instalación y arranque

### 1. Clonar el repositorio

```bash
git clone https://github.com/Is-alejandro-hub/to-do-list.git
cd to-do-list
```

### 2. Crear la base de datos

```bash
psql -U postgres -c "CREATE DATABASE todo_db;"
psql -U postgres -d todo_db -f database/01_schema.sql
```

Verifica que se crearon las 5 tablas:

```sql
\c todo_db
\dt
```

Esperado: `tasks`, `tags`, `task_tags`, `task_audit_log`, `idempotency_keys`.

### 3. Configurar el backend

```bash
cd backend
cp .env.example .env
```

Edita `.env` con tus credenciales de PostgreSQL (ver sección [Variables de entorno](#-variables-de-entorno)).

### 4. Arrancar el backend

```bash
# Desde backend/
go mod download
go run ./cmd/api
```

Salida esperada:

```
arrancando en modo development
conectando a PostgreSQL...
✅ conexión a PostgreSQL establecida
[cleanup] worker iniciado (intervalo: 1h0m0s)
🌐 servidor HTTP escuchando en :8080
```

### 5. Arrancar el frontend

En **otra terminal**:

```bash
cd frontend
npm install
ng serve
```

Salida esperada:

```
Application bundle generation complete.
➜ Local: http://localhost:4200/
```

### 6. Abrir la app

Ve a [http://localhost:4200](http://localhost:4200) en tu navegador.

**Detalle**: el frontend usa un proxy de desarrollo (`proxy.conf.json`) que redirige `/api/*` a `http://localhost:8080`. No hay problemas de CORS.

---

## 🔐 Variables de entorno

### Backend (`backend/.env`)

| Variable | Descripción | Default |
| :--- | :--- | :--- |
| `SERVER_PORT` | Puerto del servidor HTTP | `8080` |
| `APP_ENV` | Entorno (`development`, `production`) | `development` |
| `ALLOWED_ORIGINS` | Orígenes permitidos por CORS (separados por coma) | `http://localhost:4200` |
| `CLEANUP_INTERVAL_MS` | Intervalo de limpieza de claves de idempotencia (ms) | `3600000` (1h) |
| `DB_HOST` | Host de PostgreSQL | `localhost` |
| `DB_PORT` | Puerto de PostgreSQL | `5432` |
| `DB_USER` | Usuario de PostgreSQL | **requerido** |
| `DB_PASSWORD` | Contraseña de PostgreSQL | **requerido** |
| `DB_NAME` | Nombre de la base de datos | **requerido** |
| `DB_SSLMODE` | Modo SSL de PostgreSQL | `disable` |

Las variables de sistema **tienen prioridad** sobre el `.env`. Esto permite sobrescribir configuración en producción sin tocar archivos.

---

## 🌐 API REST

Base URL: `http://localhost:8080`

### Endpoints

| Método | Ruta | Descripción |
| :--- | :--- | :--- |
| `GET` | `/health` | Health check |
| `POST` | `/tasks/` | Crear tarea (idempotente con `Idempotency-Key`) |
| `GET` | `/tasks/` | Listar tareas con filtros |
| `GET` | `/tasks/{id}` | Obtener tarea por ID |
| `GET` | `/tasks/{id}/audit` | Historial de cambios de una tarea |
| `PUT` | `/tasks/{id}` | Actualizar tarea (parcial) |
| `DELETE` | `/tasks/{id}` | Soft delete |
| `POST` | `/tasks/{id}/restore` | Restaurar tarea eliminada |
| `GET` | `/tags` | Catálogo de etiquetas |

### Filtros soportados en `GET /tasks/`

| Query param | Tipo | Ejemplo |
| :--- | :--- | :--- |
| `include_deleted` | bool | `?include_deleted=true` |
| `completed` | bool | `?completed=false` |
| `priority` | enum | `?priority=HIGH` |
| `due_before` | RFC3339 | `?due_before=2026-12-31T23:59:59Z` |
| `due_after` | RFC3339 | `?due_after=2026-01-01T00:00:00Z` |
| `search` | string | `?search=ejercicio` |
| `tags` | string[] | `?tags=trabajo&tags=urgente` |

### Ejemplo de respuesta

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Hacer ejercicio",
  "description": "",
  "priority": "HIGH",
  "due_date": null,
  "completed": false,
  "created_at": "2026-01-15T15:00:00Z",
  "updated_at": "2026-01-15T15:00:00Z",
  "tags": [
    { "id": "...", "name": "rutina", "created_at": "..." },
    { "id": "...", "name": "salud", "created_at": "..." }
  ]
}
```

### Formato de errores

Todos los errores siguen el mismo formato:

```json
{
  "error": "datos inválidos",
  "details": [
    { "field": "title", "message": "el título no puede estar vacío" }
  ]
}
```

| Código | Significado |
| :--- | :--- |
| `400` | Datos de entrada inválidos |
| `404` | Recurso no encontrado |
| `405` | Método no permitido |
| `409` | Conflicto (idempotencia reutilizada con body distinto) |
| `500` | Error interno del servidor |

---

## 💾 Base de datos

### Esquema

Cinco tablas, definidas en `database/01_schema.sql`:

- **`tasks`**: tareas con prioridad, fecha de vencimiento, completado y `deleted_at` (soft delete).
- **`tags`**: catálogo global de etiquetas.
- **`task_tags`**: relación muchos-a-muchos entre tareas y etiquetas.
- **`task_audit_log`**: bitácora inmutable poblada por un trigger automático.
- **`idempotency_keys`**: cachea respuestas de POST para garantizar idempotencia.

### Trigger de auditoría

Cada operación sobre `tasks` (`INSERT`, `UPDATE`, `SOFT_DELETE`, `RESTORE`, `DELETE`) se registra automáticamente en `task_audit_log`. La aplicación **no participa** en este registro: es la base de datos la que garantiza la trazabilidad.

### Índices parciales

Los índices sobre `tasks` son parciales (`WHERE deleted_at IS NULL`). Esto los hace más pequeños y rápidos, ya que la mayoría de las consultas filtran tareas activas.

---

## 🎯 Decisiones de diseño

Las **20 decisiones técnicas** más relevantes están documentadas en [`docs/decisiones.md`](docs/decisiones.md) con contexto, alternativas consideradas y trade-offs. Algunas destacadas:

- **UUID en lugar de BIGSERIAL**: evita enumeración, funciona en sistemas distribuidos.
- **TIMESTAMPTZ en lugar de TIMESTAMP**: manejo correcto de zonas horarias.
- **ENUM nativo de PostgreSQL** para prioridad: validación en el motor.
- **Soft delete con índice parcial**: rendimiento óptimo en consultas frecuentes.
- **Trigger en lugar de código de aplicación** para auditoría: garantía absoluta.
- **Idempotencia solo en POST**: los demás métodos HTTP ya son idempotentes.
- **Punteros en `UpdateTaskInput`**: semántica correcta de PATCH (distinguir "no enviado" de "enviado con valor cero").
- **Signals en lugar de NgRx**: simplicidad y reactividad nativa en Angular 21.

---

## 🧪 Tests

### Backend

```bash
cd backend
go test ./...                    # todos los tests
go test -cover ./...             # con cobertura
go test -race ./...              # con detector de carreras (requiere cgo)
go test -v ./internal/domain/    # verbose, un paquete
```

**Cobertura actual**:
- `internal/domain`: ~100% (entidades y validaciones)
- `internal/service`: ~90% (lógica de negocio con mocks)
- `internal/handler`: sin tests unitarios (cubiertos por integración manual)

### Frontend

No hay tests automáticos en la versión actual. La verificación se hizo manualmente con una checklist funcional.

---

## 📦 Build de producción

### Frontend

```bash
cd frontend
npm ci
ng build --configuration production
```

El resultado queda en `dist/frontend/browser/` con HTML, CSS y JS minificados, hasheados y listos para servir.

**Servir localmente para verificar**:

```bash
npx http-server dist/frontend/browser -p 4300
```

### Backend

```bash
cd backend
go build -ldflags="-s -w" -o bin/api.exe ./cmd/api
```

El binario `bin/api.exe` es autocontenido: no requiere Go instalado en la máquina de destino.

### Despliegue con Nginx

En producción, Nginx sirve los archivos estáticos del frontend y actúa como reverse proxy para `/api/*` hacia el backend Go.

```nginx
server {
    listen 80;
    server_name midominio.com;

    root /var/www/todo-list/frontend/dist/frontend/browser;
    index index.html;

    # SPA fallback: redirige todas las rutas a index.html.
    location / {
        try_files $uri $uri/ /index.html;
    }

    # Reverse proxy al backend.
    location /api/ {
        proxy_pass http://localhost:8080/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

---

## 🔄 Flujo de trabajo con Git

El proyecto usa [Conventional Commits](https://www.conventionalcommits.org/) **en español**.

### Formato

```
<tipo>(<alcance>): <descripción en imperativo>
```

### Tipos usados

| Tipo | Cuándo se usa |
| :--- | :--- |
| `feat` | Nueva funcionalidad |
| `fix` | Corrección de bug |
| `refactor` | Cambio sin alterar comportamiento |
| `docs` | Solo documentación |
| `chore` | Mantenimiento (dependencias, config) |
| `style` | Formato, espacios |
| `test` | Tests |

### Alcances

`backend`, `frontend`, `db`, `docs`.

### Ejemplos

```
feat(backend): agregar endpoint de auditoría de tareas
fix(frontend): corregir indicador de filtros activos
refactor(frontend): centralizar sistema de diseño con variables SCSS
docs: agregar README principal del proyecto
chore(db): agregar migración para índice de búsqueda
```

---
---

## 📚 Documentación adicional

Además de este README, el proyecto incluye documentación especializada en la carpeta `docs/`:

- **[`docs/guia-usuario.md`](docs/guia-usuario.md)**: guía completa para usuarios finales: flujos típicos, funcionalidades avanzadas y preguntas frecuentes.
- **[`docs/decisiones.md`](docs/decisiones.md)**: 20 decisiones de diseño documentadas con contexto, alternativas y trade-offs (formato ADR).
- **[`docs/diagramas.md`](docs/diagramas.md)**: diagramas de arquitectura, flujos y modelo entidad-relación (renderizables con Mermaid).
- **[`database/01_schema.sql`](database/01_schema.sql)**: esquema completo de la base de datos con comentarios.

---

## 📄 Licencia

Proyecto desarrollado como prueba técnica. Uso libre.