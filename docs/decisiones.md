# Decisiones de diseño

Documento que registra las decisiones técnicas tomadas durante el desarrollo, con contexto, alternativas consideradas y trade-offs. Formato inspirado en [Architecture Decision Records (ADR)](https://adr.github.io/).

---

## Índice

1. [Arquitectura en capas](#1-arquitectura-en-capas)
2. [Chi como router HTTP](#2-chi-como-router-http)
3. [pgx en lugar de lib/pq](#3-pgx-en-lugar-de-libpq)
4. [UUID en lugar de BIGSERIAL](#4-uuid-en-lugar-de-bigserial)
5. [TIMESTAMPTZ en lugar de TIMESTAMP](#5-timestamptz-en-lugar-de-timestamp)
6. [ENUM nativo para prioridad](#6-enum-nativo-para-prioridad)
7. [Soft delete con índice parcial](#7-soft-delete-con-índice-parcial)
8. [Trigger para auditoría, no código de aplicación](#8-trigger-para-auditoría-no-código-de-aplicación)
9. [Sin FK en task_audit_log](#9-sin-fk-en-task_audit_log)
10. [JSONB para snapshots de auditoría](#10-jsonb-para-snapshots-de-auditoría)
11. [Idempotencia solo en POST](#11-idempotencia-solo-en-post)
12. [Idempotency-Key en lugar de hash del body](#12-idempotency-key-en-lugar-de-hash-del-body)
13. [request_hash para detectar reutilización](#13-request_hash-para-detectar-reutilización)
14. [TTL de 24h en idempotencia](#14-ttl-de-24h-en-idempotencia)
15. [Punteros en UpdateTaskInput](#15-punteros-en-updatetaskinput)
16. [Errores centinela en el dominio](#16-errores-centinela-en-el-dominio)
17. [Worker en background para limpieza](#17-worker-en-background-para-limpieza)
18. [Configuración por variables de entorno](#18-configuración-por-variables-de-entorno)
19. [Graceful shutdown](#19-graceful-shutdown)
20. [Sin procedimientos almacenados para CRUD](#20-sin-procedimientos-almacenados-para-crud)

---

## 1. Arquitectura en capas

**Contexto**: un backend Go puede organizarse de muchas formas: monolito, capas, hexagonal, CQRS, etc.

**Decisión**: arquitectura en capas con separación estricta: `handler → service → repository → domain`.

**Alternativas consideradas**:
- **Todo en main.go**: rápido de escribir, imposible de mantener.
- **Hexagonal completa**: puertos y adaptadores en todo, sobre-ingeniería para este tamaño.
- **CQRS**: separar lecturas y escrituras, útil solo con dominios complejos.

**Razones**:
- Cada capa tiene una responsabilidad única y clara.
- El dominio no depende de infraestructura. Si mañana cambias PostgreSQL por MongoDB, solo tocas `repository`.
- Los tests pueden mockear capas intermedias sin tocar la base de datos.

**Trade-offs**:
- Más archivos y más "wiring" en main.go.
- Para una prueba técnica, se podría argumentar que es sobre-ingeniería. Pero el requisito pide "proyecto robusto" y la separación es lo que permite defender cada decisión.

**Impacto**: toda la estructura del backend.

---

## 2. Chi como router HTTP

**Contexto**: Go tiene `net/http` en la stdlib, pero escribir rutas con parámetros (`/tasks/{id}`) y middleware a mano es verboso.

**Decisión**: **Chi v5**.

**Alternativas consideradas**:
- **net/http puro**: cero dependencias, pero requiere escribir matching manual de rutas.
- **Gin**: popular, pero introduce su propio `gin.Context` que no es compatible con `http.Handler`.
- **Echo**: similar a Gin, con su propia abstracción.
- **Fiber**: basado en fasthttp, no en `net/http`. Rompe compatibilidad con la stdlib.

**Razones**:
- Chi respeta `net/http` al 100%: un `chi.Mux` es un `http.Handler`.
- Cualquier middleware estándar de Go funciona sin adaptación.
- Los patrones que se aprenden con Chi son transferibles a la stdlib pura.
- Los benchmarks lo ponen en el top 3 de routers de Go.

**Trade-offs**:
- Es una dependencia externa. Pero es estable, mantenida y pequeña (~1 MB).
- Gin tiene más "baterías incluidas" (binding, validación). Chi delega eso a otras librerías. Para esta prueba, preferimos composición sobre framework.

**Impacto**: `internal/router`, `internal/handler`.

---

## 3. pgx en lugar de lib/pq

**Contexto**: los dos drivers de PostgreSQL para Go más usados son `lib/pq` y `pgx`.

**Decisión**: **pgx v5**, con `pgxpool` para el pool de conexiones.

**Alternativas consideradas**:
- **lib/pq**: el driver histórico. **Sus propios autores recomiendan migrar a pgx**.
- **database/sql + pgx stdlib**: mantiene la interfaz de `database/sql` pero pierde las optimizaciones nativas de pgx.

**Razones**:
- `lib/pq` está en modo de mantenimiento (solo bugfixes críticos).
- `pgx` soporta tipos nativos de PostgreSQL: `JSONB`, `UUID`, `ENUM`, arrays, rangos.
- `pgxpool` tiene mejor gestión de conexiones que `database/sql`.
- Soporta `LISTEN/NOTIFY`, `COPY` para bulk inserts, y otras features avanzadas.
- Mejor rendimiento en benchmarks.

**Trade-offs**:
- `pgx` no es compatible directamente con `database/sql` (aunque hay un adaptador).
- Curva de aprendizaje inicial mayor que `lib/pq`.

**Impacto**: `internal/database`, `internal/repository`.

---

## 4. UUID en lugar de BIGSERIAL

**Contexto**: los IDs de tareas pueden ser enteros secuenciales (`BIGSERIAL`) o UUIDs.

**Decisión**: **UUID v4** generado por PostgreSQL con `gen_random_uuid()`.

**Alternativas consideradas**:
- **BIGSERIAL**: 8 bytes, índices más rápidos, pero IDs predecibles (1, 2, 3...).
- **UUID v7**: ordenable por tiempo, pero PostgreSQL 13 no lo soporta nativamente. Tendríamos que generarlo en la aplicación.

**Razones**:
- **Seguridad**: los IDs secuenciales permiten enumeración. Un atacante puede recorrer `/tasks/1`, `/tasks/2`, etc. Con UUID no.
- **Distribución**: si el sistema escala a múltiples instancias del backend, cada una puede generar IDs sin coordinarse.
- **Frontend**: los UUID evitan colisiones al crear tareas localmente antes de sincronizar.

**Trade-offs**:
- 16 bytes vs 8. Los índices ocupan el doble.
- Los UUID v4 no son ordenables por inserción, causan fragmentación en índices B-tree.
- **Para esta prueba**, el volumen de datos es irrelevante. En un sistema con millones de registros por segundo, consideraríamos UUID v7 o Snowflake IDs.

**Impacto**: esquema SQL y toda la manipulación de IDs en Go.

---

## 5. TIMESTAMPTZ en lugar de TIMESTAMP

**Contexto**: PostgreSQL tiene dos tipos para timestamps: `TIMESTAMP` (sin zona) y `TIMESTAMPTZ` (con zona).

**Decisión**: **TIMESTAMPTZ** en todas las columnas de fecha.

**Alternativas consideradas**:
- **TIMESTAMP**: guarda la fecha tal cual, sin zona. Aparentemente más simple.

**Razones**:
- `TIMESTAMPTZ` guarda **siempre en UTC internamente** y convierte según la zona del cliente en lectura.
- `TIMESTAMP` no guarda zona, lo cual genera bugs cuando:
  - El servidor cambia de zona horaria (ej: migración de región).
  - Hay usuarios en distintas zonas.
  - Se compara con `NOW()` en consultas (que es `TIMESTAMPTZ`).
- **Regla general en la industria**: usa `TIMESTAMPTZ` a menos que sepas con certeza absoluta que no manejas zonas horarias.

**Trade-offs**:
- 8 bytes vs 8 bytes. No hay trade-off real de espacio.
- Más "verboso" al escribir. Pero es una vez.

**Impacto**: esquema SQL.

---

## 6. ENUM nativo para prioridad

**Contexto**: la prioridad tiene tres valores fijos: `LOW`, `MEDIUM`, `HIGH`.

**Decisión**: **ENUM nativo de PostgreSQL** (`CREATE TYPE task_priority AS ENUM (...)`).

**Alternativas consideradas**:
- **VARCHAR + CHECK**: `priority VARCHAR(10) CHECK (priority IN ('LOW', 'MEDIUM', 'HIGH'))`.
- **SMALLINT**: 1 = LOW, 2 = MEDIUM, 3 = HIGH.
- **Tabla de catálogo**: `priorities(id, name)` con FK.

**Razones**:
- **Validación garantizada por el motor**: PostgreSQL rechaza cualquier valor que no sea uno de los declarados.
- **Documentación en el esquema**: `\dT` en psql muestra los tipos definidos.
- **Rendimiento**: los ENUM se almacenan como enteros internamente (más rápido que VARCHAR).
- **Refactor fácil**: añadir un valor es `ALTER TYPE ... ADD VALUE`, no requiere migración de datos.

**Trade-offs**:
- **Cambiar el orden de los valores es engorroso**: para reordenar hay que recrear el tipo.
- **No se pueden eliminar valores**: un `DROP VALUE` no existe. Habría que recrear el tipo.
- **No se pueden reutilizar**: si tuvieras 10 columnas con la misma lista, cada una podría tener su tipo ENUM.

**Para este caso**: la lista de prioridades es fija y pequeña. El ENUM es la opción correcta.

**Impacto**: esquema SQL y tipo `Priority` en el dominio Go.

---

## 7. Soft delete con índice parcial

**Contexto**: ¿borrar tareas físicamente o solo marcarlas como eliminadas?

**Decisión**: **Soft delete** con `deleted_at TIMESTAMPTZ` (NULL = activa). Índices **parciales** sobre las columnas de búsqueda para excluir las eliminadas.

**Alternativas consideradas**:
- **DELETE físico**: simple, pero pierde historial.
- **Tabla `deleted_tasks` separada**: preserva historial pero complica las consultas.
- **Soft delete sin índices parciales**: funciona, pero los índices incluyen filas que nunca se consultan.

**Razones**:
- **Trazabilidad**: se puede auditar qué se eliminó y cuándo.
- **Reversibilidad**: `POST /tasks/{id}/restore` permite recuperar tareas eliminadas por error.
- **Coherencia con auditoría**: el trigger registra el `SOFT_DELETE` como acción específica.
- **Índices parciales son más rápidos**: al no incluir filas eliminadas, el índice es más pequeño, se actualiza más rápido y las consultas lo recorren más rápido.

**Trade-offs**:
- **Las consultas deben recordar filtrar `deleted_at IS NULL`**. Mitigado por la convención: el repositorio aplica el filtro por defecto.
- **La tabla crece indefinidamente**. Mitigado por una eventual política de purga (no implementada aquí).
- **Los índices parciales requieren PostgreSQL 9.6+**. No es un problema en este proyecto.

**Impacto**: esquema SQL, repositorio y servicio.

---

## 8. Trigger para auditoría, no código de aplicación

**Contexto**: hay que registrar cada cambio en tareas (creación, edición, eliminación, restauración).

**Decisión**: **trigger de PostgreSQL** en `INSERT`, `UPDATE` y `DELETE` sobre `tasks`, que escribe en `task_audit_log`.

**Alternativas consideradas**:
- **Código en Go**: llamar a un método `logAudit()` en cada operación del servicio.
- **Procedimiento almacenado invocado desde Go**: `CALL registrar_auditoria(...)`.
- **ORM con hooks**: usar un ORM que permita hooks antes/después de operaciones.

**Razones**:
- **Garantía absoluta**: es **imposible** modificar una tarea sin que quede registrada. Ni siquiera desde pgAdmin o `psql` se puede saltar.
- **Sin puntos de fallo humano**: en la alternativa de código Go, si alguien añade un método nuevo en el repositorio y olvida llamar a `logAudit`, la auditoría queda inconsistente.
- **Idiomático de PostgreSQL**: es la forma canónica de resolver auditoría en SQL.
- **Cero impacto en el rendimiento de la app**: el trigger corre dentro de la misma transacción, sin round-trips adicionales.

**Trade-offs**:
- **La lógica de auditoría vive en PL/pgSQL**, no en Go. Es más difícil de depurar y testear.
- **Puede sorprender a un revisor junior**: alguien que no conoce triggers podría pensar que los cambios son "mágicos".
- **Documentación obligatoria**: si no se documenta, es fácil olvidar que existe.

**Impacto**: esquema SQL, capa de datos.

---

## 9. Sin FK en task_audit_log

**Contexto**: `task_audit_log.task_id` referencia tareas, pero la relación no es una FK estándar.

**Decisión**: `task_id` es un UUID **sin FK** a `tasks`.

**Alternativas consideradas**:
- **FK con ON DELETE CASCADE**: al borrar físicamente una tarea, se borra el log.
- **FK con ON DELETE RESTRICT**: impide borrar una tarea si tiene log.
- **FK con ON DELETE SET NULL**: al borrar una tarea, el log pierde referencia.

**Razones**:
- **Preservación de trazabilidad**: si algún día se hace un `DELETE` físico (por mantenimiento, migración, etc.), el log debe sobrevivir. Un `CASCADE` destruiría el rastro.
- **Independencia del ciclo de vida**: el log es memoria histórica. No debe depender de que la tarea siga existiendo.

**Trade-offs**:
- **Sin integridad referencial**: alguien podría insertar un log con un `task_id` inválido. Mitigado porque el único que escribe en la tabla es el trigger, y el trigger solo se dispara sobre tareas existentes.
- **Consultas de "log huérfano"**: si algún día se hace limpieza, hay que tener cuidado de no borrar logs válidos.

**Impacto**: esquema SQL.

---

## 10. JSONB para snapshots de auditoría

**Contexto**: el log debe registrar el estado antes y después de cada cambio. Hay dos formas: columnas específicas (`old_title`, `new_title`, etc.) o un blob estructurado.

**Decisión**: **JSONB** con `old_data` y `new_data` como snapshots completos.

**Alternativas consideradas**:
- **Columnas específicas**: `old_title`, `old_priority`, `new_title`, `new_priority`...
- **Tabla de diffs**: registrar solo los campos que cambiaron.

**Razones**:
- **Auto-adaptativo**: si mañana se añade una columna a `tasks`, el log la captura sin cambios.
- **Snapshot completo**: reconstruir el estado histórico de una tarea es trivial.
- **Flexibilidad de consulta**: JSONB permite hacer `WHERE new_data->>'priority' = 'HIGH'` sin cambios de esquema.
- **Indexable**: se pueden crear índices GIN sobre JSONB si se necesita.

**Trade-offs**:
- **Más espacio en disco** que columnas específicas.
- **Consultas más lentas** que columnas planas (aunque JSONB es más rápido que JSON y soporta índices GIN).
- **No se pueden hacer constraints** sobre campos específicos del JSON (PostgreSQL 13 no tiene `CHECK` sobre JSONB).

**Impacto**: esquema SQL y queries de auditoría.

---

## 11. Idempotencia solo en POST

**Contexto**: la idempotencia es una propiedad del protocolo HTTP. No todos los métodos la necesitan.

**Decisión**: aplicar idempotencia **únicamente** en `POST /tasks/`.

**Alternativas consideradas**:
- **Idempotencia global**: aplicar a todos los endpoints.
- **Idempotencia configurable por endpoint**: cada handler decide.

**Razones**:
- **GET, PUT, DELETE son idempotentes por especificación HTTP**:
  - `GET` no modifica estado. Repetirlo devuelve lo mismo.
  - `PUT` reemplaza un recurso. Repetirlo con el mismo body deja el recurso en el mismo estado.
  - `DELETE` elimina un recurso. Repetirlo no cambia nada (el segundo intento da 404 pero el estado final es el mismo).
- **POST es el único no idempotente**: cada llamada crea un nuevo recurso.
- **Aplicar idempotencia a GET es absurdo**: obliga a cachear respuestas de lectura, ocupa espacio y añade latencia sin beneficio.

**Trade-offs**:
- **Requiere que el cliente envíe la cabecera**: si no lo hace, no hay protección. Pero es opcional; la app funciona sin ella.

**Impacto**: `internal/handler/task_handler.go`, tabla `idempotency_keys`.

---

## 12. Idempotency-Key en lugar de hash del body

**Contexto**: para detectar reintentos, se puede usar la clave enviada por el cliente o hashear el body y usar ese hash como clave.

**Decisión**: usar la cabecera **`Idempotency-Key`** con un UUID generado por el cliente.

**Alternativas consideradas**:
- **Hash del body**: sin cabecera adicional, el backend calcula SHA-256 del body y lo usa como clave.
- **Híbrido**: usar hash del body si no viene cabecera.

**Razones**:
- **El hash del body no distingue operaciones legítimas de reintentos**: si un usuario legítimamente quiere crear dos tareas idénticas ("Comprar leche" dos veces), el hash sería el mismo y la segunda se bloquearía. Eso es incorrecto.
- **La Idempotency-Key es la convención de la industria**: Stripe, PayPal, Square y otras APIs grandes la usan.
- **Control explícito del cliente**: el cliente decide qué operaciones deben ser idempotentes y cuáles no.

**Trade-offs**:
- **Requiere coordinación con el cliente**: el frontend debe generar un UUID por operación.
- **Ocupa más espacio** que el hash (16 bytes vs 64 bytes, pero realmente el hash es más grande).

**Impacto**: `internal/handler`, `internal/service/idempotency_service.go`.

---

## 13. request_hash para detectar reutilización

**Contexto**: si el cliente reutiliza una Idempotency-Key con un body distinto (por bug o malicia), ¿qué hacer?

**Decisión**: guardar **también** un hash SHA-256 del body (`request_hash`). Si la misma key llega con distinto hash, devolver `409 Conflict`.

**Alternativas consideradas**:
- **Ignorar el hash**: devolver siempre la respuesta cacheada, aunque el body sea distinto.
- **Sobrescribir**: tratar cada llamada como nueva operación.

**Razones**:
- **Ignorar el hash es peligroso**: si el cliente cree que creó una tarea con "Comprar pan" pero en realidad la key apuntaba a "Comprar leche", recibiría una respuesta por una tarea que nunca quiso crear.
- **Sobrescribir rompe la idempotencia**: el reintento debería devolver la misma respuesta, no crear una nueva.

**Trade-offs**:
- **Bytes adicionales en la BD**: 64 bytes por registro. Irrelevante.
- **CPU adicional**: un SHA-256 por POST. También irrelevante.

**Impacto**: `internal/service/hash.go`, tabla `idempotency_keys`.

---

## 14. TTL de 24h en idempotencia

**Contexto**: ¿cuánto tiempo guardar las claves de idempotencia?

**Decisión**: **24 horas**.

**Alternativas consideradas**:
- **Sin expiración**: la tabla crece indefinidamente.
- **1 hora**: cubre reintentos inmediatos, pero no reintentos tras horas de inactividad.
- **7 días**: cubre más casos, pero ocupa 7× más espacio.
- **Configurable por endpoint**: flexibilidad innecesaria.

**Razones**:
- **Stripe usa 24h**: es el estándar de facto de la industria.
- **Cubre el caso real**: reintentos por fallo de red ocurren en segundos o minutos, no en días.
- **Espacio manejable**: 24h de claves es un número razonable incluso con volumen alto.

**Trade-offs**:
- **Reintentos después de 24h crearán duplicados**. Aceptable: si un cliente reintenta después de un día, es un caso legítimo de nueva operación.

**Impacto**: `internal/service/idempotency_service.go`, tabla `idempotency_keys`.

---

## 15. Punteros en UpdateTaskInput

**Contexto**: en un PUT/PATCH, ¿cómo distinguir "campo no enviado" de "campo enviado con valor cero"?

**Decisión**: usar **punteros** en todos los campos de `UpdateTaskInput`.

**Alternativas consideradas**:
- **Valores planos**: `Title string`, `Completed bool`. Simple, pero rompe la semántica.
- **`sql.NullString`**: introduce acoplamiento con `database/sql`.
- **Un campo `map[string]any`**: pierde type safety.

**Razones**:
- **En Go, el "valor cero" de un string es `""` y de un bool es `false`**. Sin punteros, no se puede distinguir entre:
  - El cliente no envió `completed` → debe quedarse como estaba.
  - El cliente envió `completed=false` explícitamente → debe actualizarse a false.
- **Los punteros permiten la semántica correcta**: `nil` = no enviado, `&valor` = actualizar a ese valor.
- **Es el patrón estándar en APIs REST bien diseñadas**.

**Trade-offs**:
- **Más código**: cada acceso requiere `*input.Title` en lugar de `input.Title`.
- **Puede confundir a un junior**: "¿por qué punteros?". Requiere explicación en el README.

**Impacto**: `internal/domain/task.go`, `internal/repository/task_repository.go`, `internal/service/task_service.go`.

---

## 16. Errores centinela en el dominio

**Contexto**: el handler debe mapear errores de negocio a códigos HTTP. ¿Cómo se comunican las capas?

**Decisión**: **errores centinela** (`var ErrTaskNotFound = errors.New(...)`) + `errors.Is` para detección.

**Alternativas consideradas**:
- **Strings de error**: comparar con `err.Error() == "not found"`. Frágil.
- **Códigos numéricos**: `if err.Code == 404`. Acopla el dominio a HTTP.
- **Errores tipados como structs**: `*NotFoundError{}`. Más verboso.

**Razones**:
- **Es el patrón idiomático de Go** (ver [Go Blog: Working with Errors](https://go.dev/blog/error-handling-and-go)).
- **`errors.Is` permite envolver errores**: `fmt.Errorf("consultando tarea: %w", ErrTaskNotFound)` sigue siendo detectable con `errors.Is`.
- **El dominio no sabe qué es HTTP**: los errores son de negocio (`ErrTaskNotFound`), no de transporte (404).

**Trade-offs**:
- **Requiere que todas las capas usen `%w`** al envolver. Si alguien usa `%v`, el error deja de ser detectable.
- **Un poco de boilerplate inicial**.

**Impacto**: todo el backend.

---

## 17. Worker en background para limpieza

**Contexto**: las claves de idempotencia expiran a las 24h. ¿Cómo se limpian?

**Decisión**: **worker en background** con `time.Ticker`, que corre cada hora (configurable).

**Alternativas consideradas**:
- **`pg_cron`**: extensión de PostgreSQL que corre tareas periódicas. Añade dependencia externa.
- **Cron del sistema**: corre un script externo. Añade infraestructura.
- **Sin limpieza**: la tabla crece indefinidamente.
- **Limpieza on-demand**: al recibir un POST, primero elimina expiradas. Añade latencia a cada request.

**Razones**:
- **Sin dependencias externas**: todo el ciclo vive en el backend.
- **Ejecución inmediata al arrancar**: si el servidor estuvo caído, se limpia al volver.
- **Errores no fatales**: un fallo en la limpieza no tumba el servidor.
- **Se detiene con el servidor**: integrado con `context.Context`, sin goroutines huérfanas.

**Trade-offs**:
- **Añade una goroutine al proceso**: hay que gestionarla bien.
- **Si hay múltiples instancias del backend, todas limpian**: `DELETE` es idempotente, así que no hay problema real, pero es trabajo duplicado.

**Impacto**: `internal/scheduler/cleanup.go`, `cmd/api/main.go`.

---

## 18. Configuración por variables de entorno

**Contexto**: credenciales, puertos e intervalos deben configurarse sin recompilar.

**Decisión**: **variables de entorno** cargadas desde `.env` en desarrollo. Las variables de sistema tienen prioridad.

**Alternativas consideradas**:
- **Flags de línea de comandos**: verboso para muchas variables.
- **Archivo YAML/JSON**: un archivo más que mantener, y requiere parser.
- **Viper**: overkill para este proyecto.

**Razones**:
- **Estándar de la industria**: Docker, Kubernetes, Heroku y prácticamente todos los PaaS usan env vars.
- **`godotenv` no sobrescribe variables existentes**: en producción, las del sistema ganan. En desarrollo, el `.env` rellena los huecos.
- **Sin dependencias pesadas**: `godotenv` son ~200 líneas.

**Trade-offs**:
- **Sin tipado fuerte**: todo viene como string y hay que parsear.
- **Fácil exponer secretos en logs**: hay que tener cuidado con no loguear el `.env`.

**Impacto**: `internal/config/config.go`, `.env.example`.

---

## 19. Graceful shutdown

**Contexto**: cuando el proceso recibe SIGTERM (Docker, Kubernetes), ¿qué pasa con las peticiones en curso?

**Decisión**: **graceful shutdown** con `signal.NotifyContext` y `http.Server.Shutdown`.

**Alternativas consideradas**:
- **`os.Exit(0)` directo**: mata el proceso, corta peticiones en curso.
- **`os.Interrupt` solo**: cubre Ctrl+C, pero no SIGTERM de Docker.
- **Sin shutdown**: el sistema operativo mata el proceso.

**Razones**:
- **Docker y Kubernetes envían SIGTERM** y esperan un tiempo antes de matar con SIGKILL. Sin graceful shutdown, se pierden requests.
- **Drenaje de 10 segundos**: tiempo suficiente para que las peticiones en curso terminen.
- **Orden correcto**: primero se cierra el servidor HTTP, luego el pool de conexiones, luego el worker.
- **Idiomático de Go**: `signal.NotifyContext` es el patrón moderno.

**Trade-offs**:
- **Añade complejidad**: hay que sincronizar goroutines y esperar cierres.
- **Puede tardar hasta 10s en apagar**: a cambio, cero pérdida de datos.

**Impacto**: `cmd/api/main.go`.

---

## 20. Sin procedimientos almacenados para CRUD

**Contexto**: se podría implementar el CRUD con procedimientos almacenados en PostgreSQL en lugar de queries desde Go.

**Decisión**: **queries SQL en la capa de repositorio Go**, no procedimientos almacenados.

**Alternativas consideradas**:
- **Procedimientos almacenados para CRUD**: lógica en PL/pgSQL.
- **Un procedimiento por operación**: encapsular la lógica en la BD.

**Razones**:
- **La lógica de negocio debe vivir en el código de la aplicación**: es más fácil de testear, versionar y refactorizar.
- **PL/pgSQL es un lenguaje específico de PostgreSQL**: acopla el proyecto a un motor de BD.
- **Debugging y testing más difíciles**: probar un procedimiento almacenado es más complejo que probar una función Go.
- **El repositorio como única capa con SQL** ya encapsula suficientemente la lógica de datos.

**Cuándo SÍ usar procedimientos almacenados** (excepción):
- **Operaciones muy complejas y atómicas** que mueven grandes volúmenes de datos.
- **Auditoría automática con triggers** (que sí usamos, pero es una excepción justificada).

**Trade-offs**:
- **Rendimiento marginalmente inferior** si la red es lenta (los datos viajan al backend y vuelven). Pero en un entorno normal, es despreciable.
- **Más queries en el código Go**.

**Impacto**: `internal/repository`.

---

## Resumen de trade-offs aceptados

| Decisión | Costo aceptado | Beneficio obtenido |
| :--- | :--- | :--- |
| UUID vs BIGSERIAL | Índices 2× más grandes | Imposible enumeración |
| TIMESTAMPTZ | Nada real | Zonas horarias correctas |
| ENUM nativo | Cambios engorrosos | Validación en el motor |
| Soft delete | Filtros obligatorios | Trazabilidad, restauración |
| Trigger de auditoría | Lógica en PL/pgSQL | Garantía absoluta |
| JSONB en auditoría | Espacio y consulta | Auto-adaptación |
| Idempotencia solo POST | Cabecera del cliente | Cubre el único caso que importa |
| Punteros en Update | Código más verboso | Semántica PATCH correcta |
| Sin procedures para CRUD | Más queries en Go | Testeabilidad y portabilidad |

---

## Glosario

- **Idempotencia**: propiedad de una operación que, ejecutada N veces, produce el mismo resultado que ejecutarla una vez.
- **Soft delete**: marcar un registro como eliminado sin borrarlo físicamente.
- **Audit log**: registro histórico de cambios sobre una entidad.
- **Snapshot**: copia del estado completo de un registro en un momento dado.
- **ENUM nativo**: tipo de dato definido por el usuario con valores permitidos fijos, soportado por el motor de BD.
- **Índice parcial**: índice que solo cubre un subconjunto de filas (ej: solo las activas).
- **Trigger**: código que el motor de BD ejecuta automáticamente ante eventos sobre una tabla.
- **Graceful shutdown**: apagado ordenado que espera a que las operaciones en curso terminen.
- **Trade-off**: decisión que sacrifica algo para ganar algo más importante.

---

## Referencias

- [Go Blog: Working with Errors](https://go.dev/blog/error-handling-and-go)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [ADR GitHub](https://adr.github.io/)
- [PostgreSQL: ENUM Types](https://www.postgresql.org/docs/current/datatype-enum.html)
- [PostgreSQL: Partial Indexes](https://www.postgresql.org/docs/current/indexes-partial.html)
- [Stripe API: Idempotent Requests](https://stripe.com/docs/api/idempotent_requests)