package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Is-alejandro-hub/to-do-list/backend/internal/config"
	"github.com/Is-alejandro-hub/to-do-list/backend/internal/database"
	"github.com/Is-alejandro-hub/to-do-list/backend/internal/handler"
	"github.com/Is-alejandro-hub/to-do-list/backend/internal/repository"
	"github.com/Is-alejandro-hub/to-do-list/backend/internal/router"
	"github.com/Is-alejandro-hub/to-do-list/backend/internal/scheduler"
	"github.com/Is-alejandro-hub/to-do-list/backend/internal/service"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("error fatal: %v", err)
	}
}

func run() error {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("cargando configuración: %w", err)
	}
	log.Printf("arrancando en modo %s", cfg.Server.AppEnv)

	// Context cancelable con Ctrl+C / SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Conexión a PostgreSQL.
	log.Println("conectando a PostgreSQL...")
	pool, err := database.NewPool(ctx, cfg.DB)
	if err != nil {
		return fmt.Errorf("conectando a la base de datos: %w", err)
	}
	defer pool.Close()
	log.Println("✅ conexión a PostgreSQL establecida")

	// Ensamblado de dependencias (wiring). El orden es de abajo hacia arriba:
	// repositorios → servicios → handlers → router.
	taskRepo := repository.NewPostgresTaskRepository(pool)
	idempotencyRepo := repository.NewPostgresIdempotencyRepository(pool)

	taskService := service.NewTaskService(taskRepo)
	idempotencyService := service.NewIdempotencyService(idempotencyRepo)

	taskHandler := handler.NewTaskHandler(taskService, idempotencyService)

	httpRouter := router.New(cfg, taskHandler)

	// Worker de limpieza de claves de idempotencia expiradas.
	// Corre en background y se detiene cuando el context se cancela.
	cleanupDone := scheduler.StartIdempotencyCleanup(
		ctx,
		idempotencyService,
		time.Duration(cfg.Server.CleanupIntervalMS)*time.Millisecond,
	)

	// Servidor HTTP.
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:           httpRouter,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Arrancamos el servidor en una goroutine para poder esperar señales.
	errCh := make(chan error, 1)
	go func() {
		log.Printf("🌐 servidor HTTP escuchando en %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("servidor HTTP: %w", err)
		}
	}()

	// Esperamos por señal de apagado o por error del servidor.
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Println("señal de apagado recibida, cerrando...")
	}

	// Graceful shutdown: damos 10s para que terminen las peticiones en curso.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("error en shutdown: %w", err)
	}

	// Esperamos a que el worker termine (no bloquea más de lo necesario).
	<-cleanupDone
	log.Println("✅ apagado limpio")
	return nil
}
