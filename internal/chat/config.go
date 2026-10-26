package chat

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	DefaultModel            = "claude-sonnet-5-5"
	DefaultMaxTokens        = 2048
	DefaultRateLimitPerMin  = 20
	DefaultMaxMessageChars  = 2000
	DefaultMaxHistoryMsgs   = 12
	DefaultMaxHistoryChars  = 12000
	DefaultLLMTimeout       = 60 * time.Second
	DefaultMaxIterations    = 6
	DefaultAnthropicBaseURL = "https://api.anthropic.com"
	DefaultAnthropicVersion = "2023-06-01"
)

var (
	ErrDisabled      = errors.New("chat deshabilitado: falta ANTHROPIC_API_KEY")
	ErrMaxIterations = errors.New("se alcanzó el máximo de iteraciones de herramientas")
	ErrEmptyResponse = errors.New("el modelo no devolvió texto")
)

// Config contiene la configuración del chat cargada desde variables de entorno.
type Config struct {
	APIKey             string
	Model              string
	MaxTokens          int
	RateLimitPerMinute int
	MaxMessageChars    int
	MaxHistoryMessages int
	MaxHistoryChars    int
	LLMTimeout         time.Duration
	MaxIterations      int
	ReadDBURL          string
	AuditDBURL         string
	AnthropicBaseURL   string
}

// LoadConfig lee variables de entorno. Si existe un .env en el directorio de
// trabajo, lo carga sin sobrescribir variables ya exportadas.
func LoadConfig() Config {
	loadDotEnv(".env")

	cfg := Config{
		APIKey:             strings.TrimSpace(env("ANTHROPIC_API_KEY")),
		Model:              envDefault("CLAUDE_MODEL", DefaultModel),
		MaxTokens:          clampInt(envInt("CHAT_MAX_TOKENS", DefaultMaxTokens), 256, 8192),
		RateLimitPerMinute: clampInt(envInt("CHAT_RATE_LIMIT", DefaultRateLimitPerMin), 1, 600),
		MaxMessageChars:    clampInt(envInt("CHAT_MAX_MESSAGE_CHARS", DefaultMaxMessageChars), 100, 8000),
		MaxHistoryMessages: clampInt(envInt("CHAT_MAX_HISTORY_MESSAGES", DefaultMaxHistoryMsgs), 0, 30),
		MaxHistoryChars:    clampInt(envInt("CHAT_MAX_HISTORY_CHARS", DefaultMaxHistoryChars), 0, 40000),
		LLMTimeout:         clampDuration(envDuration("CHAT_LLM_TIMEOUT", DefaultLLMTimeout), 10*time.Second, 180*time.Second),
		MaxIterations:      DefaultMaxIterations,
		ReadDBURL:          strings.TrimSpace(env("CHAT_DB_URL")),
		AuditDBURL:         strings.TrimSpace(env("CHAT_AUDIT_DB_URL")),
		AnthropicBaseURL:   envDefault("ANTHROPIC_BASE_URL", DefaultAnthropicBaseURL),
	}
	return cfg
}

func env(key string) string {
	return strings.TrimSpace(strings.Trim(os.Getenv(key), `"'`))
}

func envDefault(key, fallback string) string {
	if v := env(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := env(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("chat: valor inválido para %s=%q; usando %d", key, v, fallback)
		return fallback
	}
	return n
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := env(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		log.Printf("chat: duración inválida para %s=%q; usando %s", key, v, fallback)
		return fallback
	}
	return d
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func clampDuration(v, min, max time.Duration) time.Duration {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// newPool crea un pool separado del pool principal de la aplicación.
func newPool(ctx context.Context, url string, maxConns int32, appName string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("URL de base de datos inválida: %w", err)
	}
	cfg.MaxConns = maxConns
	cfg.MinConns = 1
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 30 * time.Second
	if cfg.ConnConfig.RuntimeParams == nil {
		cfg.ConnConfig.RuntimeParams = map[string]string{}
	}
	cfg.ConnConfig.RuntimeParams["application_name"] = appName

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("no se pudo crear el pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("no hay respuesta de la base de datos: %w", err)
	}
	return pool, nil
}

// loadDotEnv implementa una carga mínima de .env, sin dependencias externas.
// No sobrescribe variables ya presentes en el entorno.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(strings.Trim(value, `"'`))
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, value)
	}
}
