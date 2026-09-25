# To-Do List Full Stack

Aplicación de gestión de tareas con backend en Go, frontend en Angular 21 y base de datos PostgreSQL.

Proyecto desarrollado como prueba técnica full-stack. Incluye CRUD completo, idempotencia de creación, soft delete, auditoría automática, etiquetas, prioridades, fechas de vencimiento y filtros de búsqueda.

---

## 📋 Tabla de contenidos

- [Stack tecnológico](#-stack-tecnológico)
- [Arquitectura](#-arquitectura)
- [Estructura del proyecto](#-estructura-del-proyecto)
- [Requisitos previos](#-requisitos-previos)
- [Instalación](#-instalación)
- [Variables de entorno](#-variables-de-entorno)
- [API REST](#-api-rest)
- [Base de datos](#-base-de-datos)
- [Funcionalidades adicionales](#-funcionalidades-adicionales)
- [Decisiones de diseño](#-decisiones-de-diseño)
- [Tests](#-tests)
- [Commits](#-commits)

---

## 🛠 Stack tecnológico

| Capa | Tecnología | Versión |
| :--- | :--- | :--- |
| Frontend | Angular | 21 |
| Backend | Go | 1.25 |
| Base de datos | PostgreSQL | 13+ |
| Router HTTP | Chi | v5 |
| Driver PostgreSQL | pgx | v5 |
| Gestión de configuración | godotenv | v1 |

---

## 🏗 Arquitectura

El backend sigue una **arquitectura en capas** inspirada en los principios de Clean Architecture y arquitectura hexagonal:

```
┌─────────────────────────────────────────────────────────────┐
│                     Cliente (Angular)                       │
└──────────────────────────┬──────────────────────────────────┘
                           │ HTTP / JSON
┌──────────────────────────▼──────────────────────────────────┐
│  Handler    → traduce HTTP ↔ dominio                        │
│  Service    → reglas de negocio                             │
│  Repository → acceso a datos (único lugar con SQL)          │
│  Domain     → entidades y errores de negocio                │
└──────────────────────────┬──────────────────────────────────┘
                           │ pgx
┌──────────────────────────▼──────────────────────────────────┐
│                     PostgreSQL                              │
│  - tasks             - tags           - task_tags           │
│  - task_audit_log    - idempotency_keys                     │
└─────────────────────────────────────────────────────────────┘
```

**Principio clave**: cada capa solo conoce a la inmediatamente inferior. El dominio no sabe que existe HTTP ni SQL. El repositorio es el único lugar donde hay sentencias SQL.

---

## 📁 Estructura del proyecto

```
todo-list/
├── backend/
│   ├── cmd/api/              # Punto de entrada
│   ├── internal/
│   │   ├── config/           # Carga de variables de entorno
│   │   ├── database/         # Pool de conexiones a PostgreSQL
│   │   ├── domain/           # Entidades y errores de negocio
│   │   ├── handler/          # Endpoints HTTP
│   │   ├── repository/       # Acceso a datos (SQL)
│   │   ├── router/           # Configuración de rutas y middleware
│   │   ├── scheduler/        # Worker de limpieza en background
│   │   └── service/          # Lógica de negocio
│   ├── .env.example
│   └── go.mod
├── database/
│   └── 01_schema.sql         # Esquema completo de la BD
├── docs/
│   └── diagramas.md          # Diagrama entidad-relación
└── frontend/                 # Angular 21 (Fase 4)
```

---

## ✅ Requisitos previos

- **Go 1.25** o superior
- **PostgreSQL 13** o superior
- **Node.js 22 LTS** o superior (para el frontend)
- **Angular CLI 21** (`npm install -g @angular/cli@21`)
- **Git**

---

## 🚀 Instalación

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

Verifica que se hayan creado las 5 tablas:

```sql
\dt
```

Esperado: `tasks`, `tags`, `task_tags`, `task_audit_log`, `idempotency_keys`.

### 3. Configurar el backend

```bash
cd backend
cp .env.example .env
```

Edita `.env` con tus credenciales de PostgreSQL:

```env
SERVER_PORT=8080
APP_ENV=development
ALLOWED_ORIGINS=http://localhost:4200
CLEANUP_INTERVAL_MS=3600000

DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=tu_password_real
DB_NAME=todo_db
DB_SSLMODE=disable
```

### 4. Arrancar el backend

```bash
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

El backend está listo en `http://localhost:8080`.

### 5. Frontend (Fase 4)

Pendiente. Se documentará cuando se implemente.

---

## 🔐 Variables de entorno

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

Las variables de sistema **siempre tienen prioridad** sobre `.env`. Esto permite sobrescribir configuraciones en producción sin tocar archivos.

---

## 🌐 API REST

Base URL: `http://localhost:8080`

### Endpoints

| Método | Ruta | Descripción |
| :--- | :--- | :--- |
| `GET` | `/health` | Health check |
| `POST` | `/tasks/` | Crear tarea |
| `GET` | `/tasks/` | Listar tareas con filtros |
| `GET` | `/tasks/{id}` | Obtener tarea por ID |
| `PUT` | `/tasks/{id}` | Actualizar tarea |
| `DELETE` | `/tasks/{id}` | Soft delete de tarea |
| `POST` | `/tasks/{id}/restore` | Restaurar tarea eliminada |
| `GET` | `/tags` | Listar catálogo de etiquetas |

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

### Idempotencia

`POST /tasks/` soporta la cabecera `Idempotency-Key`:

```bash
curl -X POST http://localhost:8080/tasks/ \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: 550e8400-e29b-41d4-a716-446655440000" \
  -d '{"title":"Hacer ejercicio","priority":"HIGH"}'
```

- Si es la primera vez que se envía esa clave, se crea la tarea.
- Si se reenvía la misma clave con el mismo body, se devuelve la respuesta cacheada (misma tarea, mismo `id`) y la cabecera `Idempotency-Replayed: true`.
- Si se reenvía la misma clave con un body distinto, se devuelve `409 Conflict`.

Las claves se guardan durante **24 horas** y se limpian automáticamente con un worker en background.

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

El esquema completo está en `database/01_schema.sql`. Cinco tablas:

- **`tasks`**: tareas con prioridad, fecha de vencimiento, completado y `deleted_at` (soft delete).
- **`tags`**: catálogo global de etiquetas.
- **`task_tags`**: relación muchos-a-muchos entre tareas y etiquetas.
- **`task_audit_log`**: bitácora inmutable poblada por un trigger automático.
- **`idempotency_keys`**: cachea respuestas de POST para garantizar idempotencia.

### Trigger de auditoría

Cada operación sobre `tasks` (INSERT, UPDATE, SOFT_DELETE, RESTORE, DELETE) se registra automáticamente en `task_audit_log`. La aplicación **no participa** en este registro: es la base de datos la que garantiza la trazabilidad completa.

### Índices parciales

Los índices sobre `tasks` son parciales (`WHERE deleted_at IS NULL`). Esto los hace más pequeños y rápidos, ya que la mayoría de las consultas filtran tareas activas.

---

## ✨ Funcionalidades adicionales

Además del CRUD básico, el proyecto incluye:

### Feature 1: Gestión avanzada de tareas

- **Etiquetas (tags)**: relación muchos-a-muchos para categorizar tareas. Reutilizables entre tareas.
- **Fechas de vencimiento y filtros**: cada tarea puede tener `due_date`, y el listado soporta filtros por rango de fechas, estado, prioridad, etiquetas y búsqueda textual.
- **Prioridad**: enum nativo de PostgreSQL (`LOW`, `MEDIUM`, `HIGH`). El listado ordena por prioridad y luego por fecha de vencimiento.

### Feature 2: Integridad y trazabilidad de datos

- **Soft delete**: las tareas eliminadas se marcan con `deleted_at` en lugar de borrarse físicamente. Se pueden restaurar.
- **Audit log**: cada cambio sobre una tarea queda registrado automáticamente por un trigger de PostgreSQL, con snapshot antes/después en JSONB.

### Bonus: Idempotencia de creación

El endpoint `POST /tasks/` es idempotente mediante la cabecera `Idempotency-Key`. Reintentos por fallos de red no crean duplicados.

---

## 🎯 Decisiones de diseño

Las decisiones más relevantes están documentadas en [`docs/decisiones.md`](docs/decisiones.md). Algunas destacadas:

- **UUID en lugar de BIGSERIAL** para IDs: evita enumeración, funciona en sistemas distribuidos.
- **TIMESTAMPTZ en lugar de TIMESTAMP**: manejo correcto de zonas horarias.
- **ENUM nativo de PostgreSQL** para prioridad: validación en el motor.
- **Soft delete con índice parcial**: rendimiento óptimo en consultas frecuentes.
- **Trigger en lugar de código de aplicación** para auditoría: garantía absoluta, imposible de saltar.
- **Idempotencia solo en POST**: los demás métodos HTTP ya son idempotentes por especificación.

---

## 🧪 Tests

```bash
cd backend
go test ./...
```

Para ejecutar con race detector:

```bash
go test -race ./...
```

Para cobertura:

```bash
go test -cover ./...
```

---

## 📝 Commits

El proyecto usa [Conventional Commits](https://www.conventionalcommits.org/) en español:

```
<tipo>(<alcance>): <descripción en imperativo>
```

Tipos usados: `feat`, `fix`, `refactor`, `docs`, `chore`, `style`, `test`.

Alcances: `backend`, `frontend`, `db`, `docs`.

Ejemplo:

```
feat(backend): agregar endpoint de listado con filtros dinámicos
```

---

## 📦 Build de producción

### Frontend

```bash
cd frontend
npm ci                                    # instala dependencias exactas del lock
ng build --configuration production       # genera dist/frontend/browser/

## 📄 Licencia

Proyecto desarrollado como prueba técnica. Uso libre.