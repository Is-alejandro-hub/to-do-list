package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/Is-alejandro-hub/to-do-list/backend/internal/domain"
)

// errorResponse es el formato uniforme de error que devolvemos al cliente.
// Un solo formato para todos los errores = frontend más simple.
type errorResponse struct {
	Error   string                    `json:"error"`
	Details []*domain.ValidationError `json:"details,omitempty"`
}

// writeJSON serializa cualquier valor a JSON y lo escribe con el status dado.
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload != nil {
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			// Si falla la serialización, ya enviamos el header. Solo logueamos.
			log.Printf("error serializando respuesta: %v", err)
		}
	}
}

// writeError mapea errores de dominio a códigos HTTP.
// Esta es la ÚNICA función que conoce el mapeo error → status.
// Si mañana añades un error de dominio nuevo, solo tocas aquí.
func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrTaskNotFound),
		errors.Is(err, domain.ErrTagNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})

	case errors.Is(err, domain.ErrInvalidInput):
		// Si el error concreto es un ValidationError, extraemos los detalles.
		var vErr *domain.ValidationError
		if errors.As(err, &vErr) {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Error:   "datos inválidos",
				Details: []*domain.ValidationError{vErr},
			})
			return
		}
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})

	case errors.Is(err, domain.ErrIdempotencyConflict):
		writeJSON(w, http.StatusConflict, errorResponse{Error: err.Error()})

	case errors.Is(err, domain.ErrDuplicateTag):
		writeJSON(w, http.StatusConflict, errorResponse{Error: err.Error()})

	default:
		// Error no clasificado → 500. Logueamos el detalle real,
		// pero al cliente le devolvemos un mensaje genérico.
		// NUNCA exponer stack traces ni detalles internos en producción.
		log.Printf("error interno no clasificado: %v", err)
		writeJSON(w, http.StatusInternalServerError,
			errorResponse{Error: "error interno del servidor"})
	}
}

// decodeJSON lee el body, valida que sea JSON y lo deserializa en dst.
// Limita el tamaño del body para evitar ataques DoS.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	// MaxBytesReader corta la lectura si el body supera 1MB.
	// Sin esto, un cliente malicioso podría enviar un body de GB.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields() // Rechaza campos que no estén en el struct.

	if err := dec.Decode(dst); err != nil {
		return domain.NewValidationError("body", "JSON inválido: "+err.Error())
	}
	return nil
}
