package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	Auth        AuthConfig
	LLM         LLMConfig
	Voice       VoiceConfig
	Email       EmailConfig
	Queue       QueueConfig
	RateLimit   RateLimitConfig
	Environment string
}

type ServerConfig struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	URL string
}

type RateLimitConfig struct {
	RequestsPerMinute int
	Burst             int
}

type LLMConfig struct {
	Primary    LLMProvider
	Local      LLMProvider
	Embeddings LLMProvider
	Fallback   LLMProvider
}

type LLMProvider struct {
	Provider string
	Model    string
	APIKey   string
	BaseURL  string
}

type VoiceConfig struct {
	Provider string
	Model    string
	APIKey   string
	LiveKit  LiveKitConfig
}

type LiveKitConfig struct {
	URL       string
	APIKey    string
	APISecret string
}

type EmailConfig struct {
	Provider      string
	TenantID      string
	ClientID      string
	ClientSecret  string
	CheckInterval time.Duration
	Folders       []string
}

type QueueConfig struct {
	RedisURL    string
	Concurrency int
	RetryDelay  time.Duration
}

type AuthConfig struct {
	PasswordHash string        // bcrypt hash of the single user password
	JWTSecret    string        // HMAC signing secret for JWT
	JWTExpiry    time.Duration // Token validity duration
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnvInt("SERVER_PORT", 8080),
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", ""),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: 5 * time.Minute,
		},
		Redis: RedisConfig{
			URL: getEnv("REDIS_URL", ""),
		},
		Auth: AuthConfig{
			PasswordHash: getEnv("AUTH_PASSWORD_HASH", ""),
			JWTSecret:    getEnv("AUTH_JWT_SECRET", ""),
			JWTExpiry:    24 * time.Hour,
		},
		LLM: LLMConfig{
			Primary: LLMProvider{
				Provider: "openai",
				Model:    getEnv("OPENAI_MODEL", "gpt-4o"),
				APIKey:   getEnv("OPENAI_API_KEY", ""),
			},
			Local: LLMProvider{
				Provider: "ollama",
				Model:    getEnv("OLLAMA_MODEL", "qwen2.5:latest"),
				BaseURL:  getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
			},
			Embeddings: LLMProvider{
				Provider: "ollama",
				Model:    getEnv("OLLAMA_EMBED_MODEL", "mxbai-embed-large"),
				BaseURL:  getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
			},
			Fallback: LLMProvider{
				Provider: "anthropic",
				Model:    getEnv("ANTHROPIC_MODEL", "claude-sonnet-4"),
				APIKey:   getEnv("ANTHROPIC_API_KEY", ""),
			},
		},
		Voice: VoiceConfig{
			Provider: "openai_realtime",
			Model:    getEnv("OPENAI_REALTIME_MODEL", "gpt-4o-realtime-preview"),
			APIKey:   getEnv("OPENAI_API_KEY", ""),
			LiveKit: LiveKitConfig{
				URL:       getEnv("LIVEKIT_WS_URL", "ws://localhost:7880"),
				APIKey:    getEnv("LIVEKIT_API_KEY", ""),
				APISecret: getEnv("LIVEKIT_API_SECRET", ""),
			},
		},
		Email: EmailConfig{
			Provider:      "microsoft_graph",
			TenantID:      getEnv("MS_TENANT_ID", ""),
			ClientID:      getEnv("MS_CLIENT_ID", ""),
			ClientSecret:  getEnv("MS_CLIENT_SECRET", ""),
			CheckInterval: 30 * time.Minute,
			Folders:       parseFolders(getEnv("EMAIL_FOLDERS", "Inbox")),
		},
		Queue: QueueConfig{
			RedisURL:    getEnv("REDIS_URL", ""),
			Concurrency: getEnvInt("QUEUE_CONCURRENCY", 5),
			RetryDelay:  5 * time.Second,
		},
		RateLimit: RateLimitConfig{
			RequestsPerMinute: getEnvInt("RATE_LIMIT_RPM", 60),
			Burst:             getEnvInt("RATE_LIMIT_BURST", 10),
		},
		Environment: getEnv("APP_ENV", "development"),
	}
}

// Validate checks that all required configuration is present.
// Call after Load() in main.go.
func (c *Config) Validate() error {
	if c.Auth.JWTSecret == "" {
		return errors.New("config: JWT secret required")
	}
	if c.Auth.PasswordHash == "" {
		return errors.New("config: password hash required")
	}
	if c.Database.URL == "" {
		return errors.New("config: database URL required")
	}
	if c.Redis.URL == "" {
		return errors.New("config: Redis URL required")
	}
	if c.LLM.Primary.APIKey == "" {
		return errors.New("config: primary LLM API key required")
	}
	if c.LLM.Fallback.APIKey == "" {
		return errors.New("config: fallback LLM API key required")
	}
	if c.Voice.APIKey == "" {
		return errors.New("config: voice API key required")
	}
	if c.Email.TenantID == "" || c.Email.ClientID == "" || c.Email.ClientSecret == "" {
		return errors.New("config: Microsoft Graph credentials required")
	}
	if c.Voice.LiveKit.APIKey == "" || c.Voice.LiveKit.APISecret == "" {
		return errors.New("config: LiveKit credentials required")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func parseFolders(s string) []string {
	if s == "" {
		return []string{"Inbox"}
	}
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
