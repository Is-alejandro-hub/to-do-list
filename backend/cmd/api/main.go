package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Is-alejandro-hub/to-do-list/backend/internal/config"
	"github.com/Is-alejandro-hub/to-do-list/backend/internal/database"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("error fatal: %v", err)
	}
}

// run encapsula la lógica de arranque para que main() sea trivial
// y podamos hacer testing si algún día lo necesitamos.
func run() error {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("cargando configuración: %w", err)
	}

	log.Printf("arrancando en modo %s", cfg.Server.AppEnv)

	// Context que se cancela con Ctrl+C (SIGINT) o SIGTERM (Docker/K8s).
	// Todo lo que dependa de este contexto se detiene limpiamente.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("conectando a PostgreSQL...")
	pool, err := database.NewPool(ctx, cfg.DB)
	if err != nil {
		return fmt.Errorf("conectando a la base de datos: %w", err)
	}
	defer pool.Close()

	log.Println("✅ conexión a PostgreSQL establecida")

	// Ping de verificación para confirmar la salud del pool.
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		return fmt.Errorf("ping falló: %w", err)
	}
	log.Println("✅ ping exitoso")

	// Aquí irá el servidor HTTP en el siguiente paso.

	log.Println("🎉 backend listo. Ctrl+C para salir.")
	<-ctx.Done()
	log.Println("apagando limpiamente...")

	return nil
}
