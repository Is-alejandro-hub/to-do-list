package domain

import "errors"

// Errores de dominio. Son "centinelas": valores únicos que podemos
// comparar con errors.Is() desde las capas superiores.
//
// Regla: los errores de negocio viven en el dominio, no en el handler
// ni en el repositorio. Así cualquier capa puede referenciarlos sin
// crear dependencias circulares.
var (
	// ErrTaskNotFound se devuelve cuando se busca una tarea por ID
	// y no existe (o está eliminada lógicamente, según el contexto).
	ErrTaskNotFound = errors.New("tarea no encontrada")

	// ErrTagNotFound se devuelve cuando se referencia un tag inexistente
	// en una operación que exige que exista.
	ErrTagNotFound = errors.New("etiqueta no encontrada")

	// ErrInvalidInput es el error genérico de validación. El detalle
	// específico va dentro de un ValidationError (ver abajo).
	ErrInvalidInput = errors.New("datos de entrada inválidos")

	// ErrIdempotencyConflict se devuelve cuando llega una Idempotency-Key
	// que ya existe pero con un body distinto (request_hash diferente).
	// Es un 409 Conflict a nivel HTTP.
	ErrIdempotencyConflict = errors.New("clave de idempotencia reutilizada con payload distinto")

	// ErrDuplicateTag se devuelve al intentar crear un tag con nombre
	// ya existente.
	ErrDuplicateTag = errors.New("la etiqueta ya existe")
)

// ValidationError envuelve ErrInvalidInput con detalles por campo.
// Es un error tipado, así el handler puede extraer el campo y el mensaje
// y devolverlos en la respuesta JSON al frontend.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// Unwrap permite que errors.Is(err, ErrInvalidInput) funcione
// incluso cuando el error concreto es un *ValidationError.
func (e *ValidationError) Unwrap() error {
	return ErrInvalidInput
}

// NewValidationError es un constructor de conveniencia.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{Field: field, Message: message}
}
