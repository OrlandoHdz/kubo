package database

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"gopkg.in/yaml.v3" 
)

// Config estructura para mapear el YAML
type Config struct {
	Database struct {
		Url      string `yaml:"url"`
		MaxConns int32  `yaml:"max_conns"`
		MinConns int32  `yaml:"min_conns"`
	} `yaml:"database"`
}

// NuevoPool crea un pool de conexiones configurado
func NuevoPool(ctx context.Context, configPath string) (*pgxpool.Pool, error) {
	// 1. Leer el archivo YAML
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error leyendo config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error parseando yaml: %w", err)
	}

	// 2. Configurar el pool de pgx/v5
	poolConfig, err := pgxpool.ParseConfig(cfg.Database.Url)
	if err != nil {
		return nil, fmt.Errorf("url de conexión inválida: %w", err)
	}

	poolConfig.MaxConns = cfg.Database.MaxConns
	poolConfig.MinConns = cfg.Database.MinConns

	// 3. Crear el pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("no se pudo crear el pool: %w", err)
	}

	// Verificar conexión inicial
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("no hay respuesta de la base de datos: %w", err)
	}

	return pool, nil
}