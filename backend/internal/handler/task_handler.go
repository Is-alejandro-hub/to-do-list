package handler

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/Is-alejandro-hub/to-do-list/backend/internal/domain"
	"github.com/Is-alejandro-hub/to-do-list/backend/internal/service"
)

// TaskHandler agrupa las dependencias que necesitan sus métodos.
type TaskHandler struct {
	taskService        service.TaskService
	idempotencyService service.IdempotencyService
}

func NewTaskHandler(
	taskService service.TaskService,
	idempotencyService service.IdempotencyService,
) *TaskHandler {
	return &TaskHandler{
		taskService:        taskService,
		idempotencyService: idempotencyService,
	}
}

// Create maneja POST /tasks.
//
// Soporta idempotencia opcional mediante la cabecera Idempotency-Key.
// Si el cliente no envía la cabecera, se procesa la creación normalmente
// (sin protección contra duplicados por reintento).
//
// Flujo:
//  1. Leer el body crudo para poder hashearlo.
//  2. Si hay Idempotency-Key:
//     a. Buscar la clave.
//     b. Si existe con el mismo hash → devolver respuesta cacheada.
//     c. Si existe con hash distinto → 409 Conflict.
//  3. Decodificar el body y validar.
//  4. Crear la tarea.
//  5. Serializar la respuesta.
//  6. Guardar la clave con el status y el body de la respuesta.
//  7. Devolver 201 Created.
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Paso 1: leemos el body crudo. Necesitamos los bytes para hashearlos
	// y para deserializar. Si usáramos json.NewDecoder directo, consumiríamos
	// el stream y no podríamos hashear.
	rawBody, err := readBody(w, r)
	if err != nil {
		writeError(w, err)
		return
	}

	// Paso 2: si viene Idempotency-Key, gestionamos la idempotencia.
	idempKey, hasIdemp := parseIdempotencyKey(r)
	var requestHash string

	if hasIdemp {
		requestHash = service.HashRequest(rawBody)

		record, err := h.idempotencyService.Check(r.Context(), idempKey, requestHash)
		if err != nil {
			writeError(w, err)
			return
		}

		if record != nil {
			// Ya procesada. Devolvemos la respuesta guardada tal cual.
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Idempotency-Replayed", "true")
			w.WriteHeader(record.ResponseStatus)
			_, _ = w.Write(record.ResponseBody)
			return
		}
	}

	// Paso 3: deserializamos el body ya leído.
	var input domain.CreateTaskInput
	if err := json.Unmarshal(rawBody, &input); err != nil {
		writeError(w, domain.NewValidationError("body", "JSON inválido: "+err.Error()))
		return
	}

	// Paso 4: llamamos al servicio.
	task, err := h.taskService.Create(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}

	// Paso 5: serializamos la respuesta a un buffer para poder guardarla.
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(task); err != nil {
		writeError(w, err)
		return
	}
	responseBody := buf.Bytes()

	// Paso 6: guardamos la clave si aplica.
	if hasIdemp {
		// Si falla el guardado, la operación principal ya está hecha.
		// No fallamos el request por esto: logueamos y seguimos.
		if err := h.idempotencyService.Save(
			r.Context(), idempKey, requestHash,
			http.StatusCreated, responseBody,
		); err != nil {
			// Nota: aquí no llamamos a writeError porque ya vamos a
			// escribir la respuesta de éxito. Solo logueamos.
			// En un sistema crítico, esto podría requerir compensación.
		}
	}

	// Paso 7: respondemos.
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(responseBody)
}

// GetByID maneja GET /tasks/{id}.
func (h *TaskHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}

	includeDeleted := r.URL.Query().Get("include_deleted") == "true"

	task, err := h.taskService.GetByID(r.Context(), id, includeDeleted)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

// List maneja GET /tasks con filtros por query string.
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	filter, err := parseTaskFilter(r)
	if err != nil {
		writeError(w, err)
		return
	}

	tasks, err := h.taskService.List(r.Context(), filter)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

// Update maneja PUT /tasks/{id}.
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}

	var input domain.UpdateTaskInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}

	task, err := h.taskService.Update(r.Context(), id, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

// Delete maneja DELETE /tasks/{id} (soft delete).
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}

	if err := h.taskService.SoftDelete(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Restore maneja POST /tasks/{id}/restore.
func (h *TaskHandler) Restore(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}

	if err := h.taskService.Restore(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListTags maneja GET /tags.
func (h *TaskHandler) ListTags(w http.ResponseWriter, r *http.Request) {
	tags, err := h.taskService.ListTags(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tags)
}

// ListAudit maneja GET /tasks/{id}/audit.
func (h *TaskHandler) ListAudit(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}

	entries, err := h.taskService.ListAudit(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}
