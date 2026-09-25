package router

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/Is-alejandro-hub/to-do-list/backend/internal/config"
	"github.com/Is-alejandro-hub/to-do-list/backend/internal/handler"
)

// New construye el router con todas las rutas y middleware configurados.
//
// Orden de middleware (de fuera hacia adentro):
//  1. RequestID     → asigna ID único a cada petición
//  2. Recover       → captura pánicos
//  3. Logging       → loguea la petición completa
//  4. CORS          → gestiona preflight y headers
//  5. Compress      → comprime respuestas (de chi)
//  6. Timeout       → corta peticiones demasiado lentas
//
// El orden importa: RequestID antes que Logging para que los logs tengan ID;
// Recover antes que Logging para que un pánico se loguee igual; CORS antes
// que los handlers para que el preflight no llegue al handler.
func New(cfg *config.Config, taskHandler *handler.TaskHandler) http.Handler {
	r := chi.NewRouter()

	// Middleware base.
	r.Use(chimw.RequestID) // chi ya inyecta un request ID; lo usamos.
	r.Use(handler.RequestIDMiddleware)
	r.Use(handler.RecoverMiddleware)
	r.Use(handler.LoggingMiddleware)
	r.Use(handler.CORSMiddleware(cfg.Server.AllowedOrigins))
	r.Use(chimw.Compress(5)) // gzip nivel 5: buen balance CPU/ratio.
	r.Use(chimw.Timeout(30 * time.Second))

	// Health check: útil para Kubernetes, Docker y para ti mismo.
	r.Get("/health", healthCheck)

	// Rutas de la API.
	r.Route("/tasks", func(r chi.Router) {
		r.Post("/", taskHandler.Create)
		r.Get("/", taskHandler.List)
		r.Get("/{id}", taskHandler.GetByID)
		r.Get("/{id}/audit", taskHandler.ListAudit)
		r.Put("/{id}", taskHandler.Update)
		r.Delete("/{id}", taskHandler.Delete)
		r.Post("/{id}/restore", taskHandler.Restore)

	})

	r.Get("/tags", taskHandler.ListTags)

	// 404 y 405 personalizados con formato JSON uniforme.
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"ruta no encontrada"}`))
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte(`{"error":"método no permitido"}`))
	})

	return r
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
