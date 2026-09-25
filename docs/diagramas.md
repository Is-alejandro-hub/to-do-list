# Diagramas de arquitectura

Documentación visual de la arquitectura, flujos y modelo de datos del proyecto.

Todos los diagramas usan [Mermaid](https://mermaid.js.org/) y se renderizan automáticamente en GitHub, GitLab y VS Code con la extensión *Markdown Preview Mermaid Support*.

## Índice

1. [Arquitectura de capas del backend](#1-arquitectura-de-capas-del-backend)
2. [Secuencia HTTP de POST con idempotencia](#2-secuencia-http-de-post-con-idempotencia)
3. [Flujo de datos del frontend](#3-flujo-de-datos-del-frontend)
4. [Máquina de estados del listado](#4-máquina-de-estados-del-listado)
5. [Flujo de auditoría por trigger](#5-flujo-de-auditoría-por-trigger)
6. [Modelo entidad-relación](#6-modelo-entidad-relación)

---

## 1. Arquitectura de capas del backend

Cómo fluye una petición HTTP a través de las capas. Cada capa solo conoce a la inmediatamente inferior.

```mermaid
flowchart TB
    subgraph Cliente["🌐 Cliente (navegador / curl / Postman)"]
        Req[Petición HTTP]
    end

    subgraph Backend["⚙️ Backend (Go)"]
        direction TB
        Router["Router + Middleware<br/>(Chi)<br/>──<br/>request id · logging<br/>recover · CORS"]
        Handler["Handler<br/>──<br/>parsea HTTP<br/>llama al servicio<br/>mapea errores"]
        Service["Service<br/>──<br/>reglas de negocio<br/>validación<br/>idempotencia"]
        Repo["Repository<br/>──<br/>ÚNICO lugar con SQL<br/>pgx + pgxpool"]
    end

    subgraph DB["💾 PostgreSQL"]
        Tables["tasks · tags · task_tags<br/>task_audit_log · idempotency_keys"]
        Trigger["🔧 Trigger de auditoría<br/>(automático)"]
    end

    Req --> Router
    Router --> Handler
    Handler --> Service
    Service --> Repo
    Repo --> Tables
    Tables -.->|AFTER INSERT/UPDATE/DELETE| Trigger
    Trigger -.->|INSERT| Tables

    classDef domain fill:#e6f3ff,stroke:#3182ce,color:#1a202c
    classDef infra fill:#fff5e6,stroke:#d69e2e,color:#1a202c
    classDef storage fill:#e6ffe6,stroke:#38a169,color:#1a202c

    class Handler,Service domain
    class Router,Repo infra
    class Tables,Trigger storage
```

**Lectura clave**:
- El **dominio** (`Handler`, `Service`) no sabe qué es SQL ni HTTP.
- El **repositorio** es el único punto de contacto con PostgreSQL.
- El **trigger** corre dentro de la misma transacción, sin round-trips adicionales.

---

## 2. Secuencia HTTP de POST con idempotencia

El flujo más complejo del sistema: crear una tarea con protección contra duplicados.

```mermaid
sequenceDiagram
    autonumber
    participant C as Cliente (Angular)
    participant P as Proxy (ng serve / Nginx)
    participant H as Handler
    participant I as IdempotencyService
    participant DB as PostgreSQL

    C->>C: generateUUID()<br/>(Idempotency-Key)
    C->>P: POST /api/tasks/<br/>Header: Idempotency-Key<br/>Body: {title, priority, ...}
    P->>H: POST /tasks/

    Note over H: 1. Leer body crudo
    Note over H: 2. Calcular SHA-256 del body

    H->>I: Check(key, request_hash)
    I->>DB: SELECT * FROM idempotency_keys<br/>WHERE key = $1

    alt Clave no existe (primera vez)
        DB-->>I: nil
        I-->>H: (nil, nil)

        H->>H: Deserializar body a CreateTaskInput
        H->>H: taskService.Create(input)

        Note over H: Servicio valida,<br/>deduplica tags,<br/>llama al repositorio

        H->>DB: INSERT INTO tasks +<br/>INSERT INTO task_tags +<br/>INSERT INTO idempotency_keys
        DB-->>H: tarea creada + clave guardada

        H->>C: 201 Created<br/>Body: {tarea}
        Note over C: Nueva tarea

    else Clave existe, mismo hash (replay)
        DB-->>I: record encontrado
        I-->>H: (record, nil)

        H->>C: 201 Created<br/>Header: Idempotency-Replayed: true<br/>Body: respuesta cacheada
        Note over C: Misma respuesta<br/>Sin duplicado

    else Clave existe, hash distinto (conflicto)
        DB-->>I: record con otro hash
        I-->>H: error ErrIdempotencyConflict

        H->>C: 409 Conflict<br/>Body: {error}
        Note over C: Cliente cometió un error
    end
```

**Puntos clave**:
- **El hash SHA-256 del body** detecta reutilización de la misma key con distinto payload.
- **La respuesta se guarda como JSON crudo** en `idempotency_keys.response_body`. Si el cliente reintenta, se devuelve tal cual.
- **El header `Idempotency-Replayed: true`** comunica al cliente que la respuesta es un replay.

---

## 3. Flujo de datos del frontend

Cómo se conectan los componentes, el store y el servicio HTTP.

```mermaid
flowchart LR
    subgraph Componentes["🎨 Componentes"]
        TL[TaskList]
        TF[TaskForm]
        TH[TaskHistory]
        TC[TaskCard<br/>presentacional]
    end

    subgraph Estado["🧠 Estado (Signals)"]
        Store[TaskStore<br/>──<br/>#tasks · #tags<br/>#loading · #error<br/>#saving]
    end

    subgraph Servicios["🌐 Servicios"]
        SVC[TaskService<br/>──<br/>HttpClient]
    end

    subgraph Backend["⚙️ API REST"]
        API["/api/tasks<br/>/api/tags"]
    end

    TL -->|lee signals| Store
    TC -->|recibe task vía @input| TL

    TF -->|store.create/update| Store
    TH -->|taskService.getAudit| SVC

    Store -->|taskService.list/create/update| SVC
    SVC -->|HTTP| API

    API -.->|Observable| SVC
    SVC -.->|subscribe| Store
    Store -.->|notifica signals| TL
    Store -.->|notifica signals| TF

    classDef comp fill:#e6f3ff,stroke:#3182ce,color:#1a202c
    classDef state fill:#fff5e6,stroke:#d69e2e,color:#1a202c
    classDef srv fill:#e6ffe6,stroke:#38a169,color:#1a202c

    class TL,TF,TH,TC comp
    class Store state
    class SVC srv
```

**Reglas de oro**:
- Los componentes **nunca** llaman a `HttpClient` directamente. Siempre pasan por el store o el servicio.
- El store **nunca** muta el array localmente después de una operación: recarga desde el backend para mantener coherencia.
- Los componentes **solo leen signals** y llaman a métodos. Nunca escriben al estado directamente.

---

## 4. Máquina de estados del listado

Los estados visuales que puede tener la página `TaskList`.

```mermaid
stateDiagram-v2
    [*] --> Cargando: ngOnInit

    Cargando --> Vacio: respuesta vacía<br/>sin filtros
    Cargando --> ConTareas: respuesta con tareas
    Cargando --> SinResultados: respuesta vacía<br/>con filtros activos
    Cargando --> Error: fallo en la petición

    ConTareas --> Cargando: cambio de filtro<br/>o mutación
    SinResultados --> Cargando: cambio de filtro
    SinResultados --> ConTareas: limpiar filtros
    Vacio --> Cargando: crear primera tarea
    Error --> Cargando: reintentar

    ConTareas --> [*]: navegar fuera
    Vacio --> [*]: navegar fuera
    SinResultados --> [*]: navegar fuera
    Error --> [*]: navegar fuera
```

**Decisión clave**: los estados `Vacio` y `SinResultados` se diferencian visualmente:
- **Vacio**: "No hay tareas todavía" → call to action "Crear primera tarea".
- **SinResultados**: "Sin resultados" → call to action "Limpiar filtros".

Esta distinción evita que el usuario piense que perdió sus datos cuando en realidad los filtros no coinciden.

---

## 5. Flujo de auditoría por trigger

Cómo se registra automáticamente cada cambio, sin que la aplicación Go participe.

```mermaid
sequenceDiagram
    autonumber
    participant App as Aplicación Go
    participant DB as PostgreSQL
    participant T as Trigger<br/>fn_tasks_audit()
    participant Log as task_audit_log

    Note over App: UPDATE tasks<br/>SET completed = true<br/>WHERE id = $1

    App->>DB: UPDATE tasks (con SET LOCAL app.current_user)
    activate DB

    DB->>T: BEFORE UPDATE<br/>(fn_set_updated_at)
    T->>DB: NEW.updated_at := NOW()

    DB->>T: AFTER UPDATE<br/>(fn_tasks_audit)
    activate T

    Note over T: 1. Determina acción<br/>(INSERT/UPDATE/SOFT_DELETE/RESTORE)

    T->>T: to_jsonb(OLD) → old_data
    T->>T: to_jsonb(NEW) → new_data
    T->>T: current_setting('app.current_user')<br/>o current_user

    T->>Log: INSERT INTO task_audit_log<br/>(task_id, action, old_data, new_data, changed_by)
    deactivate T

    DB-->>App: UPDATE completado
    deactivate DB

    Note over App: La app NO sabe que se registró<br/>el cambio en el log

    Note over Log: 📜 El rastro queda para siempre<br/>incluso si la tarea se borra físicamente
```

**Puntos clave**:
- El trigger corre **dentro de la misma transacción** del `UPDATE`. Si el trigger falla, el `UPDATE` también falla. Esto es lo que garantiza la consistencia.
- El usuario se obtiene de `SET LOCAL app.current_user` si la app lo seteó. Si no, cae a `current_user` de PostgreSQL (útil para cambios manuales en pgAdmin).
- **La aplicación Go no participa**. Es imposible saltarse la auditoría desde el código.

---

## 6. Modelo entidad-relación

Estructura de datos de la aplicación.

```mermaid
erDiagram
    TASKS {
        uuid id PK
        varchar title
        text description
        task_priority priority
        timestamptz due_date
        boolean completed
        timestamptz deleted_at
        timestamptz created_at
        timestamptz updated_at
    }

    TAGS {
        uuid id PK
        varchar name UK
        timestamptz created_at
    }

    TASK_TAGS {
        uuid task_id PK,FK
        uuid tag_id PK,FK
    }

    TASK_AUDIT_LOG {
        bigserial id PK
        uuid task_id
        varchar action
        jsonb old_data
        jsonb new_data
        varchar changed_by
        timestamptz changed_at
    }

    IDEMPOTENCY_KEYS {
        uuid key PK
        varchar request_hash
        int response_status
        jsonb response_body
        timestamptz created_at
        timestamptz expires_at
    }

    TASKS ||--o{ TASK_TAGS : "tiene"
    TAGS ||--o{ TASK_TAGS : "aplicado a"
    TASKS ||--o{ TASK_AUDIT_LOG : "auditada por"
```

**Detalles sutiles**:
- **`TASK_AUDIT_LOG` no tiene FK** hacia `tasks`. Es deliberado: si se borra físicamente una tarea, la auditoría sobrevive.
- **`IDEMPOTENCY_KEYS` es independiente**: no referencia a `tasks` porque su propósito es puramente técnico.
- **`TASK_TAGS` usa PK compuesta** `(task_id, tag_id)`. No puede existir la misma dupla dos veces, lo cual es una forma de idempotencia a nivel de BD.
- **`deleted_at` permite soft delete** con índice parcial.

---

## Cómo ver estos diagramas

### En VS Code

1. Instala la extensión **Markdown Preview Mermaid Support** (bierner.markdown-mermaid).
2. Abre este archivo.
3. Presiona `Ctrl+Shift+V` para ver la vista previa.

### En GitHub

GitHub renderiza Mermaid **automáticamente** en archivos `.md`. Solo tienes que abrir el archivo en el navegador.

### En la presentación

Si expones este archivo en pantalla compartida, los diagramas se ven claramente. Cada uno cuenta una parte de la historia del sistema.