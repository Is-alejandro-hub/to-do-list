package handler

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Is-alejandro-hub/to-do-list/backend/internal/domain"
)

// readBody lee el body completo con un límite de tamaño.
// Devuelve los bytes crudos, listos para hashear y deserializar.
func readBody(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, domain.NewValidationError("body", "no se pudo leer el body")
	}
	if len(body) == 0 {
		return nil, domain.NewValidationError("body", "el body está vacío")
	}
	return body, nil
}

// parseIdempotencyKey extrae y valida la cabecera Idempotency-Key.
// Devuelve (uuid, true) si la cabecera existe y es un UUID válido.
// Devuelve (uuid.Nil, false) si la cabecera no está.
// Devuelve error si existe pero no es un UUID válido.
func parseIdempotencyKey(r *http.Request) (uuid.UUID, bool) {
	raw := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if raw == "" {
		return uuid.Nil, false
	}
	key, err := uuid.Parse(raw)
	if err != nil {
		// Cabecera presente pero inválida. Tratamos como si no estuviera
		// para no romper al cliente; podríamos rechazar con 400 si fuéramos
		// estrictos. Aquí elegimos la opción permisiva.
		return uuid.Nil, false
	}
	return key, true
}

// parseUUIDParam extrae un UUID de la URL. Ej: /tasks/{id}.
func parseUUIDParam(r *http.Request, name string) (uuid.UUID, error) {
	raw := chi.URLParam(r, name)
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, domain.NewValidationError(name, "UUID inválido")
	}
	return id, nil
}

// parseTaskFilter construye un TaskFilter a partir de la query string.
func parseTaskFilter(r *http.Request) (domain.TaskFilter, error) {
	q := r.URL.Query()
	filter := domain.TaskFilter{}

	if v := q.Get("include_deleted"); v != "" {
		b, err := parseBool(v)
		if err != nil {
			return filter, domain.NewValidationError("include_deleted", "debe ser true o false")
		}
		filter.IncludeDeleted = &b
	}

	if v := q.Get("completed"); v != "" {
		b, err := parseBool(v)
		if err != nil {
			return filter, domain.NewValidationError("completed", "debe ser true o false")
		}
		filter.Completed = &b
	}

	if v := q.Get("priority"); v != "" {
		p := domain.Priority(strings.ToUpper(v))
		if !p.IsValid() {
			return filter, domain.NewValidationError("priority", "LOW, MEDIUM o HIGH")
		}
		filter.Priority = &p
	}

	if v := q.Get("due_before"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return filter, domain.NewValidationError("due_before", "formato RFC3339 requerido")
		}
		filter.DueBefore = &t
	}

	if v := q.Get("due_after"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return filter, domain.NewValidationError("due_after", "formato RFC3339 requerido")
		}
		filter.DueAfter = &t
	}

	if v := q.Get("search"); v != "" {
		filter.Search = v
	}

	// tags puede venir repetido: ?tags=trabajo&tags=urgente
	if tags := q["tags"]; len(tags) > 0 {
		filter.TagNames = tags
	}

	return filter, nil
}

func parseBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "true", "1":
		return true, nil
	case "false", "0":
		return false, nil
	}
	return false, errors.New("valor booleano inválido")
}
