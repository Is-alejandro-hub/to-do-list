package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/Is-alejandro-hub/to-do-list/backend/internal/service"
)

// StartIdempotencyCleanup lanza un worker en background que elimina
// periódicamente las claves de idempotencia expiradas.
//
// Decisiones de diseño:
//
//  1. Se ejecuta UNA VEZ al arrancar, antes del primer tick. Si el servidor
//     estuvo caído toda la noche, hay claves expiradas esperando; no tiene
//     sentido esperar una hora para limpiarlas.
//
//  2. Los errores NO son fatales. Un fallo al limpiar no debe tumbar el
//     servidor. Logueamos y esperamos al siguiente tick.
//
//  3. Se detiene cuando el context se cancela. Al recibir SIGTERM, la
//     goroutine muere limpiamente con el resto del servidor.
//
//  4. Devuelve un channel que se cierra cuando el worker termina. Esto
//     permite al llamador (main) esperar la finalización si lo desea.
//
// El parámetro `interval` controla cada cuánto corre la limpieza.
func StartIdempotencyCleanup(
	ctx context.Context,
	svc service.IdempotencyService,
	interval time.Duration,
) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)

		log.Printf("[cleanup] worker iniciado (intervalo: %s)", interval)

		// Ejecución inmediata al arrancar.
		runCleanup(ctx, svc)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("[cleanup] worker detenido")
				return
			case <-ticker.C:
				runCleanup(ctx, svc)
			}
		}
	}()

	return done
}

// runCleanup ejecuta una limpieza puntual. Extraído en su propia función
// para que sea testeable de forma aislada si algún día se quiere.
func runCleanup(ctx context.Context, svc service.IdempotencyService) {
	// Timeout propio: si PostgreSQL está lento, no queremos que el worker
	// quede bloqueado indefinidamente. 30 segundos es más que suficiente
	// para un DELETE indexado.
	cleanupCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	deleted, err := svc.CleanupExpired(cleanupCtx)
	if err != nil {
		log.Printf("[cleanup] error eliminando claves expiradas: %v", err)
		return
	}

	if deleted > 0 {
		log.Printf("[cleanup] %d claves de idempotencia eliminadas", deleted)
	}
}
