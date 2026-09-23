package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config agrupa toda la configuración de la aplicación.
// Los tags `mapstructure` no son necesarios aquí porque no usamos Viper,
// pero el patrón general (una struct única) sí es el mismo.
type Config struct {
	Server ServerConfig
	DB     DBConfig
}

type ServerConfig struct {
	Port   int
	AppEnv string
}

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

// Load lee el archivo .env (si existe) y las variables de entorno del sistema.
// Las variables reales del sistema SIEMPRE tienen prioridad sobre el .env,
// lo cual es clave para despliegues en producción (Docker, Kubernetes, etc.).
func Load() (*Config, error) {
	// godotenv.Load NO sobrescribe variables ya definidas en el entorno.
	// Si el archivo no existe, simplemente lo ignora.
	_ = godotenv.Load()

	serverPort, err := strconv.Atoi(getEnv("SERVER_PORT", "8080"))
	if err != nil {
		return nil, fmt.Errorf("SERVER_PORT inválido: %w", err)
	}

	dbPort, err := strconv.Atoi(getEnv("DB_PORT", "5432"))
	if err != nil {
		return nil, fmt.Errorf("DB_PORT inválido: %w", err)
	}

	cfg := &Config{
		Server: ServerConfig{
			Port:   serverPort,
			AppEnv: getEnv("APP_ENV", "development"),
		},
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     dbPort,
			User:     requireEnv("DB_USER"),
			Password: requireEnv("DB_PASSWORD"),
			Name:     requireEnv("DB_NAME"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
	}

	return cfg, nil
}

// DSN construye la cadena de conexión para pgx.
// Formato: postgres://user:password@host:port/dbname?sslmode=...
func (c DBConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode,
	)
}

// getEnv devuelve el valor de la variable o el default si no existe.
func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return defaultValue
}

// requireEnv devuelve el valor de la variable o entra en pánico si falta.
// La usamos para secretos: preferimos fallar al arrancar que fallar después.
func requireEnv(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		panic(fmt.Sprintf("variable de entorno requerida no definida: %s", key))
	}
	return value
}
