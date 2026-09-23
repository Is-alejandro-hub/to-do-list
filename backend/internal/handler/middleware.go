package handler

import (
	"context"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
)

// contextKey es un tipo propio (no string) para las claves del context.
// Es una buena práctica de Go: evita colisiones con claves de otros paquetes.
type contextKey string

const requestIDKey contextKey = "request_id"

// RequestIDMiddleware genera un ID único por petición y lo inyecta en el
// contexto. Se propaga en el header X-Request-ID de la respuesta para que
// el cliente pueda correlacionar logs con peticiones.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.NewString()
		}

		// Inyectamos en el contexto. Todo lo que dependa del contexto
		// (incluidos los servicios en llamadas posteriores) puede leerlo.
		ctx := context.WithValue(r.Context(), requestIDKey, reqID)
		w.Header().Set("X-Request-ID", reqID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID extrae el request ID del contexto. Devuelve "" si no existe.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// statusRecorder envuelve un ResponseWriter para capturar el status code.
// http.ResponseWriter no expone el status una vez escrito; hay que interceptarlo.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK // Write implica 200 si no se llamó WriteHeader.
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// LoggingMiddleware registra método, path, status, duración y request ID
// de cada petición. Es el mínimo vital para poder depurar en producción.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: 0}

		next.ServeHTTP(rec, r)

		log.Printf(
			"%s %s %d %dB %s req_id=%s",
			r.Method, r.URL.Path, rec.status, rec.bytes,
			time.Since(start).Round(time.Millisecond),
			GetRequestID(r.Context()),
		)
	})
}

// RecoverMiddleware captura pánicos en cualquier handler y los convierte
// en 500. Sin este middleware, un pánico derribaría el servidor entero.
//
// Importante: NO exponemos el detalle del pánico en la respuesta. Logueamos
// el stack completo pero el cliente recibe un mensaje genérico.
func RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("PÁNICO recuperado en %s %s: %v",
					r.Method, r.URL.Path, rec)
				log.Printf("stack trace:\n%s", debug.Stack())

				writeJSON(w, http.StatusInternalServerError,
					errorResponse{Error: "error interno del servidor"})
			}
		}()

		next.ServeHTTP(w, r)
	})
}
