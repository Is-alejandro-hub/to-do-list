package service

import "testing"

func TestHashRequest_Deterministic(t *testing.T) {
	body := []byte(`{"title":"Hacer ejercicio","priority":"HIGH"}`)

	h1 := HashRequest(body)
	h2 := HashRequest(body)

	if h1 != h2 {
		t.Errorf("el hash no es determinista: %q vs %q", h1, h2)
	}
}

func TestHashRequest_DifferentBodiesDiffer(t *testing.T) {
	body1 := []byte(`{"title":"Hacer ejercicio"}`)
	body2 := []byte(`{"title":"Hacer otra cosa"}`)

	if HashRequest(body1) == HashRequest(body2) {
		t.Errorf("dos bodies distintos dieron el mismo hash")
	}
}

func TestHashRequest_Format(t *testing.T) {
	h := HashRequest([]byte("test"))

	// SHA-256 en hexadecimal son 64 caracteres.
	if len(h) != 64 {
		t.Errorf("len(hash) = %d, se esperaban 64 caracteres hex", len(h))
	}
}

func TestHashRequest_EmptyBody(t *testing.T) {
	// El hash de un body vacío no debe entrar en pánico.
	h := HashRequest([]byte{})
	if h == "" {
		t.Errorf("el hash de un body vacío no debe ser vacío")
	}
}
