package domain

import (
	"errors"
	"testing"
)

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{Field: "title", Message: "no puede estar vacío"}

	got := err.Error()
	want := "title: no puede estar vacío"

	if got != want {
		t.Errorf("Error() = %q, se esperaba %q", got, want)
	}
}

func TestValidationError_Unwrap(t *testing.T) {
	err := NewValidationError("title", "vacío")

	// errors.Is debe poder detectar ErrInvalidInput a través del wrapper.
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("errors.Is(err, ErrInvalidInput) = false, se esperaba true")
	}
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("field_x", "mensaje_y")

	if err.Field != "field_x" {
		t.Errorf("Field = %q, se esperaba %q", err.Field, "field_x")
	}
	if err.Message != "mensaje_y" {
		t.Errorf("Message = %q, se esperaba %q", err.Message, "mensaje_y")
	}
}

func TestErrorSentinels_AreDistinct(t *testing.T) {
	// Verificación de seguridad: los errores centinela no deben ser iguales
	// entre sí. Un typo que los iguale pasaría desapercibido sin este test.
	sentinels := []error{
		ErrTaskNotFound,
		ErrTagNotFound,
		ErrInvalidInput,
		ErrIdempotencyConflict,
		ErrDuplicateTag,
	}

	for i := 0; i < len(sentinels); i++ {
		for j := i + 1; j < len(sentinels); j++ {
			if errors.Is(sentinels[i], sentinels[j]) {
				t.Errorf("los errores en posiciones %d y %d son iguales: %v, %v",
					i, j, sentinels[i], sentinels[j])
			}
		}
	}
}
