package domain

import (
	"time"

	"github.com/google/uuid"
)

// IdempotencyRecord representa el resultado de un POST ya procesado.
// Se guarda para poder devolver la misma respuesta ante reintentos.
type IdempotencyRecord struct {
	Key            uuid.UUID
	RequestHash    string
	ResponseStatus int
	ResponseBody   []byte // JSON crudo de la respuesta
	CreatedAt      time.Time
	ExpiresAt      time.Time
}
