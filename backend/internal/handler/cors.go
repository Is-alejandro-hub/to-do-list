package handler

import (
	"net/http"
	"strings"
)

// CORSMiddleware configura los headers de Cross-Origin Resource Sharing.
//
// allowedOrigins es la lista de orígenes permitidos. En desarrollo será
// ["http://localhost:4200"]; en producción, el dominio real de la app.
//
// Nota: no usamos el paquete cors de chi porque escribir CORS bien es
// sorprendentemente sutil y queremos que sea explícito y auditable.
func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	// Precomputamos un set para lookup O(1).
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[strings.TrimSpace(o)] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if _, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Vary", "Origin")
			}

			// Preflight: el navegador pregunta antes de hacer la petición real.
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods",
					"GET, POST, PUT, DELETE, PATCH, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers",
					"Content-Type, Authorization, Idempotency-Key, X-Request-ID")
				w.Header().Set("Access-Control-Max-Age", "86400") // 24h
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
