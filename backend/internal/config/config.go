package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	LLM      LLMConfig
	Voice    VoiceConfig
	Email    EmailConfig
	Queue    QueueConfig
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

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnvInt("SERVER_PORT", 8080),
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", "postgres://myjob:myjob_dev@localhost:5432/myjob?sslmode=disable"),
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
		},
		Redis: RedisConfig{
			URL: getEnv("REDIS_URL", "redis://localhost:6379"),
		},
		LLM: LLMConfig{
			Primary: LLMProvider{
				Provider: "openai",
				Model:    "gpt-4o",
				APIKey:   getEnv("OPENAI_API_KEY", ""),
			},
			Local: LLMProvider{
				Provider: "ollama",
				Model:    "qwen2.5:latest",
				BaseURL:  getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
			},
			Embeddings: LLMProvider{
				Provider: "ollama",
				Model:    "mxbai-embed-large",
				BaseURL:  getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
			},
			Fallback: LLMProvider{
				Provider: "anthropic",
				Model:    "claude-sonnet-4-20250514",
				APIKey:   getEnv("ANTHROPIC_API_KEY", ""),
			},
		},
		Voice: VoiceConfig{
			Provider: "openai_realtime",
			Model:    "gpt-4o-realtime-preview",
			APIKey:   getEnv("OPENAI_API_KEY", ""),
			LiveKit: LiveKitConfig{
				URL:       getEnv("LIVEKIT_WS_URL", "ws://localhost:7880"),
				APIKey:    getEnv("LIVEKIT_API_KEY", "devkey"),
				APISecret: getEnv("LIVEKIT_API_SECRET", "devsecret"),
			},
		},
		Email: EmailConfig{
			Provider:      "microsoft_graph",
			TenantID:      getEnv("MS_TENANT_ID", ""),
			ClientID:      getEnv("MS_CLIENT_ID", ""),
			ClientSecret:  getEnv("MS_CLIENT_SECRET", ""),
			CheckInterval: 30 * time.Minute,
			Folders:       []string{"Inbox"},
		},
		Queue: QueueConfig{
			RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),
			Concurrency: 5,
			RetryDelay:  5 * time.Second,
		},
	}
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
