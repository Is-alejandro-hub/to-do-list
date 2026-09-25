# Guía de usuario

Cómo usar la aplicación To-Do List en el día a día.

## Índice

1. [Inicio rápido](#-inicio-rápido)
2. [La interfaz](#-la-interfaz)
3. [Flujos típicos](#-flujos-típicos)
4. [Funcionalidades avanzadas](#-funcionalidades-avanzadas)
5. [Preguntas frecuentes](#-preguntas-frecuentes)
6. [Atajos de teclado](#-atajos-de-teclado)

---

## 🚀 Inicio rápido

Al abrir [http://localhost:4200](http://localhost:4200) verás el listado de tus tareas. Si es la primera vez, verás un mensaje de bienvenida invitándote a crear tu primera tarea.

**Crear una tarea en 3 pasos**:

1. Haz clic en **"+ Nueva tarea"** en la esquina superior derecha.
2. Rellena el título (obligatorio) y, opcionalmente, descripción, prioridad, fecha de vencimiento y etiquetas.
3. Haz clic en **"Crear tarea"**. Serás redirigido al listado y verás tu nueva tarea.

**Fin.** Eso es todo para empezar.

---

## 🎨 La interfaz

### Header

En la parte superior encontrarás:

- **Logo "✓ To-Do List"**: clic para volver al listado principal.
- **"Listado"**: navega al listado de tareas.
- **"+ Nueva tarea"**: accede al formulario de creación.

El header se mantiene fijo mientras haces scroll, con un efecto de vidrio esmerilado.

### Listado de tareas

Cada tarea se muestra como una tarjeta con:

```
┌──────────────────────────────────────────────────────────────┐
│  ◯  Hacer ejercicio                                     [×]  │
│     Cardio 30 minutos                                        │
│     [ALTA]  📅 Vence: 15 ene 2026  #salud  #rutina           │
│                                                              │
│                                    [Historial] [Editar] [×]  │
└──────────────────────────────────────────────────────────────┘
```

- **◯ círculo a la izquierda**: marca la tarea como completada. Vuelve a hacer clic para desmarcarla.
- **Badge de prioridad**: `ALTA` (rojo), `MEDIA` (naranja) o `BAJA` (verde).
- **Fecha de vencimiento**: aparece si le asignaste una. Si la fecha ya pasó y la tarea no está completada, se muestra **en rojo**.
- **Etiquetas**: chips con `#nombre`.
- **Botones**: `Historial`, `Editar`, `Eliminar`.

Cuando una tarea está completada, se atenúa y su título aparece tachado.

### Filtros

Debajo del header, una barra con:

- **Buscador**: filtra por título o descripción (con debounce de 300 ms).
- **Selector de prioridad**: todas, alta, media o baja.
- **"Mostrar eliminadas"**: muestra también las tareas que marcaste como eliminadas.
- **"Filtros avanzados"**: despliega opciones adicionales.

Dentro de "Filtros avanzados":

- **Vence desde / Vence hasta**: rango de fechas.
- **Etiquetas**: chips seleccionables para filtrar por etiquetas específicas.
- **Limpiar filtros**: resetea todo.

Cuando aplicas al menos un filtro, aparece un **badge azul** con el número total de filtros activos junto a "Filtros avanzados".

---

## 🎯 Flujos típicos

### Crear una tarea simple

1. Clic en **"+ Nueva tarea"**.
2. Escribe **"Comprar leche"** en el título.
3. Clic en **"Crear tarea"**.

### Crear una tarea con toda la información

1. Clic en **"+ Nueva tarea"**.
2. Rellena:
   - **Título**: "Preparar presentación del proyecto"
   - **Descripción**: "Slides de 15 minutos con demo en vivo"
   - **Prioridad**: `ALTA`
   - **Fecha de vencimiento**: el viernes de esta semana
   - **Etiquetas**: `trabajo, urgente, presentación`
3. Clic en **"Crear tarea"**.

### Marcar una tarea como completada

En el listado, haz clic en el **círculo** a la izquierda del título. La tarea se atenúa y el título se tacha.

Para desmarcarla, haz clic de nuevo.

### Editar una tarea existente

1. En el listado, clic en **"Editar"** en la tarjeta.
2. Modifica lo que necesites.
3. Clic en **"Guardar cambios"**.

**Nota**: en modo edición aparece un checkbox **"Marcar como completada"**. En modo creación no aparece porque las tareas nuevas siempre empiezan sin completar.

### Eliminar una tarea

1. Clic en **"Eliminar"** en la tarjeta.
2. Confirma el diálogo nativo del navegador.

**La tarea no se borra físicamente**. Se marca como eliminada y puedes recuperarla.

### Recuperar una tarea eliminada

1. Marca **"Mostrar eliminadas"** en la barra de filtros.
2. Localiza la tarea (aparece con fondo gris y borde punteado).
3. Clic en **"Restaurar"**.

### Ver el historial de una tarea

1. Clic en **"Historial"** en la tarjeta de la tarea.
2. Verás una **línea de tiempo** con todos los cambios:

```
─────────────────────────────────────────────────────
│ [CREADA]    15 ene 2026, 10:30      Título inicial: Hacer ejercicio
├─────────────────────────────────────────────────────
│ [ACTUALIZADA]  15 ene 2026, 11:45   por app_user
│              completed: No → Sí
│              priority: Media → Alta
├─────────────────────────────────────────────────────
│ [ELIMINADA]  16 ene 2026, 09:15     por app_user
└─────────────────────────────────────────────────────
```

Cada entrada muestra:
- **Tipo de cambio** (badge con color).
- **Fecha y hora**.
- **Usuario** que hizo el cambio.
- **Diff de los campos modificados** (antes → después).

### Filtrar tareas por etiqueta

1. Abre **"Filtros avanzados"**.
2. En la sección **"Etiquetas"**, haz clic en los chips de las etiquetas que te interesen.
3. Los chips seleccionados se colorean de azul.
4. Puedes seleccionar varias: el listado muestra tareas que tengan **al menos una** de las etiquetas seleccionadas.

### Buscar tareas por texto

Escribe en el buscador. El listado se filtra automáticamente **300 ms después de que pares de escribir** (debounce). Busca tanto en el título como en la descripción.

### Filtrar por rango de fechas

1. Abre **"Filtros avanzados"**.
2. Selecciona **"Vence desde"** y/o **"Vence hasta"**.
3. El listado se actualiza.

**Ejemplo**: si seleccionas "Vence desde: 01/01/2026" y "Vence hasta: 31/01/2026", verás solo tareas con fecha de vencimiento dentro de enero.

---

## ⚡ Funcionalidades avanzadas

### Idempotencia invisible

La aplicación tiene protección contra duplicados cuando creas tareas. Si por un fallo de red se reintenta automáticamente un `POST`, **no se crean dos tareas**.

**No tienes que hacer nada**: el frontend genera un ID único por operación y el backend lo detecta.

### Auditoría automática

**Cada cambio** sobre una tarea se registra con fecha, usuario y diff. Esto incluye:

- Cambios hechos desde la app.
- Cambios hechos manualmente desde pgAdmin o `psql` (útil para debugging).

**No se puede desactivar**: es una garantía del sistema, no una funcionalidad opcional.

### Soft delete

Las tareas eliminadas **no se borran** de la base de datos. Se marcan con una fecha de eliminación (`deleted_at`). Beneficios:

- Puedes recuperarlas.
- El historial de cambios se preserva.
- La auditoría sigue mostrando cuándo se eliminaron.

### Prioridad con ordenamiento automático

El listado ordena por:

1. **Prioridad**: alta → media → baja.
2. **Fecha de vencimiento**: las más próximas primero.
3. **Fecha de creación**: las más recientes primero.

---

## ❓ Preguntas frecuentes

### ¿Los datos se guardan en la nube?

No. Todo se guarda en tu **PostgreSQL local**. Si quieres respaldar, usa `pg_dump`:

```bash
pg_dump -U postgres todo_db > backup.sql
```

### ¿Puedo usar la app en varios dispositivos?

La versión actual está pensada para desarrollo local. Para uso multi-dispositivo necesitarías desplegar el backend en un servidor accesible y configurar HTTPS.

### ¿Cómo elimino una tarea permanentemente?

**No hay opción en la UI**. Todas las eliminaciones son soft delete. Si necesitas borrar físicamente, hazlo desde `psql`:

```sql
DELETE FROM tasks WHERE id = 'uuid-aqui';
```

**Nota**: la auditoría sobrevive incluso si borras la tarea físicamente.

### ¿Por qué a veces tarda 300 ms en filtrar la búsqueda?

Es el **debounce**. En lugar de disparar una petición HTTP por cada tecla, esperamos 300 ms desde la última tecla. Esto:

- Reduce la carga del servidor.
- Evita peticiones innecesarias.
- Se siente instantáneo en la práctica.

### ¿Qué pasa si cierro el navegador a la mitad de crear una tarea?

Si ya enviaste el `POST` al backend, la tarea se crea. Si el navegador se cierra antes, no se crea nada.

Gracias a la **idempotencia**, si por un fallo de red el navegador reintenta automáticamente la operación, no se crea un duplicado.

### ¿Puedo tener tareas repetitivas?

No en esta versión. Las tareas son objetos únicos. Si quieres una tarea semanal, tendrías que crearla cada semana.

**Alternativa manual**: usa la misma etiqueta (`#semanal`) en todas tus tareas recurrentes y filtra por esa etiqueta.

### ¿Cómo veo quién cambió una tarea?

En la vista de historial. Si modificaste la tarea desde la app, verás el usuario que configuraste. Si la modificaste desde `psql`, verás `postgres` (el usuario de la base de datos).

### ¿La app funciona sin internet?

Sí, si el backend y PostgreSQL están corriendo localmente. No depende de servicios externos.

### ¿Cómo cambio las etiquetas de una tarea?

Edita la tarea. En el campo **"Etiquetas"** del formulario, cambia el texto:

```
trabajo, urgente, presentación
```

Separa las etiquetas con **comas**. El sistema las limpia y deduplica automáticamente (si escribes `trabajo, Trabajo, TRABAJO`, se guarda solo una).

---

## ⌨️ Atajos de teclado

La app soporta navegación completa por teclado:

| Tecla | Acción |
| :--- | :--- |
| `Tab` | Siguiente elemento interactivo |
| `Shift + Tab` | Elemento anterior |
| `Enter` | Activar botón o link enfocado |
| `Espacio` | Activar checkbox enfocado |
| `Esc` | Cerrar diálogo nativo del navegador |

**Recomendación**: navega con `Tab` para verificar visualmente los **anillos de foco azules** que indican qué elemento está activo.

---

## 💡 Consejos para sacarle el máximo partido

1. **Usa etiquetas consistentes**: si usas `trabajo` y `laburo` como sinónimos, tendrás dos filtros separados. Elige una convención y mantenla.

2. **Combina filtros**: puedes filtrar por "Prioridad: Alta" + "Vence hasta: 31/12/2026" + "Etiqueta: urgente" para ver **exactamente** lo que necesitas.

3. **Revisa el historial antes de restaurar una tarea eliminada**: verás por qué se eliminó y si vale la pena recuperarla.

4. **Usa la búsqueda por texto con palabras clave**: "comprar" encuentra "Comprar leche", "Comprar pan", "Comprar regalo". No necesitas recordar el título exacto.

5. **Las tareas vencidas aparecen en rojo**: son un recordatorio visual. Si ves muchas en rojo, considera priorizarlas o ajustar las fechas.

---

## 🐛 Reportar problemas

Esta es una aplicación de prueba técnica, no un producto en producción. Si encuentras un bug:

1. Abre la **consola del navegador** (F12).
2. Anota los **logs del backend** (terminal donde corre `go run ./cmd/api`).
3. Anota los **pasos exactos** para reproducir el problema.

Con esa información, cualquier desarrollador puede diagnosticar el problema rápidamente.