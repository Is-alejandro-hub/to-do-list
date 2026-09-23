package service

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashRequest devuelve el SHA-256 hexadecimal de un payload.
// Se usa para detectar reutilización de Idempotency-Key con body distinto.
//
// Nota: hashear el body crudo (bytes) es más fiel que hashear el JSON
// re-serializado. Dos representaciones distintas del mismo objeto JSON
// (por orden de claves, espacios, etc.) producirían hashes distintos.
// Eso está bien: para efectos de idempotencia, "mismo body" significa
// "byte a byte idéntico", no "semánticamente equivalente".
func HashRequest(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
