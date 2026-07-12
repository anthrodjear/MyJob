package config

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Config struct {
	Server          ServerConfig
	Database        DatabaseConfig
	Redis           RedisConfig
	Auth            AuthConfig
	LLM             LLMConfig
	Voice           VoiceConfig
	Email           EmailConfig
	Queue           QueueConfig
	Scoring         ScoringConfig
	Prompts         PromptsConfig
	RateLimit       RateLimitConfig
	AuthRateLimit   AuthRateLimitConfig
	Environment     string
}

type ServerConfig struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	CORSOrigins  []string
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

type AuthRateLimitConfig struct {
	RequestsPerMinute int
	Burst             int
}

type LLMConfig struct {
	Primary         LLMProvider
	Local           LLMProvider
	Embeddings      LLMProvider
	Fallback        LLMProvider
	EmailClassifier LLMProvider
}

type LLMProvider struct {
	Provider string
	Model    string
	APIKey   string
	BaseURL  string
	Timeout  time.Duration
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
	Concurrency int
	RetryDelay  time.Duration
}

type ScoringWeights struct {
	Skill       float64
	Experience  float64
	Location    float64
	Salary      float64
	Description float64
}

type ScoringConfig struct {
	AutoThreshold      int            // score >= this = auto apply
	ReviewThreshold    int            // score >= this = human review
	Weights            ScoringWeights // factor weights (must sum to 1.0)
	Mode               string         // "heuristic", "llm", or "hybrid" (default: "hybrid")
	HybridRejectMargin int            // heuristic margin below review threshold to auto-reject (default: 20)
}

// PromptsConfig holds all LLM prompts centralized in config.
type PromptsConfig struct {
	Scoring           PromptPair
	EmailClassifier   PromptPair
	CoverLetter       PromptPair
	ResumeTailor      PromptPair
	InterviewPrep     PromptPair
	JobExtraction     PromptPair
	FormUnderstanding PromptPair
	ResumeGeneration  PromptPair
}

// PromptPair holds a system + user prompt template.
type PromptPair struct {
	System string
	User   string
}

type AuthConfig struct {
	PasswordHash        string        // bcrypt hash of the single user password
	JWTSecret           string        // HMAC signing secret for JWT
	JWTExpiry           time.Duration // Token validity duration
	RefreshTokenExpiry  time.Duration // Refresh token validity duration (default: 7 days)
	BCryptCost          int           // bcrypt cost factor (default: 12, min: 10)
}

// IsProduction returns true if running in production environment.
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// IsDevelopment returns true if running in development environment.
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsTest returns true if running in test environment.
func (c *Config) IsTest() bool {
	return c.Environment == "test"
}

func Load() *Config {
	// Read application.yaml for prompt loading
	configPath := getEnv("CONFIG_PATH", "config/application.yaml")
	yamlData, _ := os.ReadFile(configPath) //nolint:gosec // G304: config path is from env var, not user input

	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnvInt("SERVER_PORT", 8080),
			ReadTimeout:  getEnvDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getEnvDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			CORSOrigins:  parseCommaList(getEnv("CORS_ORIGINS", "http://localhost:3000")),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", ""),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			URL: getEnv("REDIS_URL", ""),
		},
		Auth: AuthConfig{
			PasswordHash:       getEnv("AUTH_PASSWORD_HASH", ""),
			JWTSecret:          getEnv("AUTH_JWT_SECRET", ""),
			JWTExpiry:          getEnvDuration("JWT_EXPIRY", 30*time.Minute),
			RefreshTokenExpiry: getEnvDuration("REFRESH_TOKEN_EXPIRY", 7*24*time.Hour), // 7 days
			BCryptCost:         getEnvInt("BCRYPT_COST", 12),
		},
		LLM: LLMConfig{
			Primary: LLMProvider{
				Provider: getEnv("LLM_PRIMARY_PROVIDER", "openai"),
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
				Timeout:  getEnvDuration("OLLAMA_TIMEOUT", 60*time.Second),
			},
			Fallback: LLMProvider{
				Provider: getEnv("LLM_FALLBACK_PROVIDER", "anthropic"),
				Model:    getEnv("ANTHROPIC_MODEL", "claude-sonnet-4"),
				APIKey:   getEnv("ANTHROPIC_API_KEY", ""),
			},
			EmailClassifier: LLMProvider{
				Provider: getEnv("LLM_EMAIL_CLASSIFIER_PROVIDER", "ollama"),
				Model:    getEnv("OLLAMA_MODEL", "qwen2.5:latest"),
				BaseURL:  getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
				Timeout:  getEnvDuration("OLLAMA_TIMEOUT", 60*time.Second),
			},
		},
		Voice: VoiceConfig{
			Provider: getEnv("VOICE_PROVIDER", "ollama"),
			Model:    getEnv("VOICE_MODEL", "qwen2.5:latest"), //HACK-change to config no hardcodding
			APIKey:   getEnv("OPENAI_API_KEY", ""),
			LiveKit: LiveKitConfig{
				URL:       getEnv("LIVEKIT_WS_URL", "ws://localhost:7880"),
				APIKey:    getEnv("LIVEKIT_API_KEY", ""),
				APISecret: getEnv("LIVEKIT_API_SECRET", ""),
			},
		},
		Email: EmailConfig{
			Provider:      getEnv("EMAIL_PROVIDER", "microsoft_graph"),
			TenantID:      getEnv("MS_TENANT_ID", ""),
			ClientID:      getEnv("MS_CLIENT_ID", ""),
			ClientSecret:  getEnv("MS_CLIENT_SECRET", ""),
			CheckInterval: getEnvDuration("EMAIL_CHECK_INTERVAL", 30*time.Minute),
			Folders:       parseFolders(getEnv("EMAIL_FOLDERS", "Inbox")),
		},
		Queue: QueueConfig{
			Concurrency: getEnvInt("QUEUE_CONCURRENCY", 5),
			RetryDelay:  getEnvDuration("QUEUE_RETRY_DELAY", 5*time.Second),
		},
		Scoring: ScoringConfig{
			AutoThreshold:      getEnvInt("SCORING_AUTO_THRESHOLD", 95),
			ReviewThreshold:    getEnvInt("SCORING_REVIEW_THRESHOLD", 80),
			Mode:               getEnv("SCORING_MODE", "hybrid"),
			HybridRejectMargin: getEnvInt("SCORING_HYBRID_REJECT_MARGIN", 20),
			Weights: ScoringWeights{
				Skill:       getEnvFloat("SCORING_WEIGHT_SKILL", 0.35),
				Experience:  getEnvFloat("SCORING_WEIGHT_EXPERIENCE", 0.25),
				Location:    getEnvFloat("SCORING_WEIGHT_LOCATION", 0.10),
				Salary:      getEnvFloat("SCORING_WEIGHT_SALARY", 0.15),
				Description: getEnvFloat("SCORING_WEIGHT_DESCRIPTION", 0.15),
			},
		},
		RateLimit: RateLimitConfig{
			RequestsPerMinute: getEnvInt("RATE_LIMIT_RPM", 60),
			Burst:             getEnvInt("RATE_LIMIT_BURST", 10),
		},
		AuthRateLimit: AuthRateLimitConfig{
			RequestsPerMinute: getEnvInt("AUTH_RATE_LIMIT_RPM", 5),
			Burst:             getEnvInt("AUTH_RATE_LIMIT_BURST", 3),
		},
		Prompts:     LoadPromptsFromYAML(yamlData),
		Environment: getEnv("APP_ENV", "development"),
	}
	return cfg
}

// Validate checks that all required configuration is present.
// Call after Load() in main.go.
func (c *Config) Validate() error {
	// Environment validation
	switch c.Environment {
	case "development", "staging", "production", "test":
	default:
		return fmt.Errorf("config: invalid APP_ENV: %s (must be development, staging, production, or test)", c.Environment)
	}

	// Auth validation
	if c.Auth.JWTSecret == "" {
		return errors.New("config: JWT secret required")
	}
	if len(c.Auth.JWTSecret) < 32 {
		return errors.New("config: JWT secret must be at least 32 characters")
	}
	// PasswordHash is now optional — setup flow creates the user if not set.
	// When set, validates format for backward compatibility.
	if c.Auth.PasswordHash != "" && !strings.HasPrefix(c.Auth.PasswordHash, "$2") {
		return errors.New("config: invalid bcrypt hash format")
	}
	// BCrypt cost validation
	if c.Auth.BCryptCost < bcrypt.MinCost {
		return fmt.Errorf("config: bcrypt cost must be at least %d, got %d", bcrypt.MinCost, c.Auth.BCryptCost)
	}
	if c.Auth.BCryptCost > bcrypt.MaxCost {
		return fmt.Errorf("config: bcrypt cost must not exceed %d, got %d", bcrypt.MaxCost, c.Auth.BCryptCost)
	}
	// Refresh token expiry validation
	if c.Auth.RefreshTokenExpiry <= 0 {
		return errors.New("config: refresh token expiry must be positive")
	}
	if c.Auth.RefreshTokenExpiry <= c.Auth.JWTExpiry {
		return errors.New("config: refresh token expiry must be longer than JWT expiry")
	}

	// Infrastructure validation
	if c.Database.URL == "" {
		return errors.New("config: database URL required")
	}
	if c.Database.MaxOpenConns <= 0 {
		return errors.New("config: MaxOpenConns must be positive")
	}
	if c.Database.MaxIdleConns < 0 {
		return errors.New("config: MaxIdleConns must be non-negative")
	}
	if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		return errors.New("config: MaxIdleConns must not exceed MaxOpenConns")
	}
	if c.Database.ConnMaxLifetime < 0 {
		return errors.New("config: ConnMaxLifetime must be non-negative")
	}
	if c.Redis.URL == "" {
		return errors.New("config: Redis URL required")
	}

	// LLM validation — provider-based (only validate what's configured)
	if c.LLM.Primary.Provider == "openai" && c.LLM.Primary.APIKey == "" {
		return errors.New("config: OpenAI API key required when primary provider is openai")
	}
	if c.LLM.Fallback.Provider == "anthropic" && c.LLM.Fallback.APIKey == "" {
		return errors.New("config: Anthropic API key required when fallback provider is anthropic")
	}

	// Voice validation — only if provider is set
	if c.Voice.Provider == "openai_realtime" && c.Voice.APIKey == "" {
		return errors.New("config: OpenAI API key required for voice provider")
	}
	if c.Voice.LiveKit.APIKey == "" || c.Voice.LiveKit.APISecret == "" {
		return errors.New("config: LiveKit credentials required")
	}

	// Email validation — only if provider is microsoft_graph and credentials are provided
	if c.Email.Provider == "microsoft_graph" {
		if c.Email.TenantID != "" || c.Email.ClientID != "" || c.Email.ClientSecret != "" {
			if c.Email.TenantID == "" || c.Email.ClientID == "" || c.Email.ClientSecret == "" {
				return errors.New("config: Microsoft Graph credentials required when provider is microsoft_graph")
			}
		}
		// If provider is microsoft_graph but no credentials at all, that's OK - it'll just fail at runtime
	}

	// Scoring validation
	if c.Scoring.AutoThreshold <= c.Scoring.ReviewThreshold {
		return errors.New("config: scoring auto threshold must be greater than review threshold")
	}
	if c.Scoring.HybridRejectMargin < 0 {
		return errors.New("config: scoring hybrid reject margin must be non-negative")
	}
	if c.Scoring.HybridRejectMargin >= c.Scoring.ReviewThreshold {
		return errors.New("config: scoring hybrid reject margin must be less than review threshold")
	}
	validModes := map[string]struct{}{"heuristic": {}, "llm": {}, "hybrid": {}}
	if _, ok := validModes[c.Scoring.Mode]; !ok {
		return errors.New("config: scoring mode must be one of: heuristic, llm, hybrid")
	}
	if err := c.validateScoringWeights(); err != nil {
		return err
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

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return fallback
}

func parseFolders(s string) []string {
	if s == "" {
		return []string{"Inbox"}
	}
	parts := strings.Split(s, ",")
	var folders []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			folders = append(folders, part)
		}
	}
	if len(folders) == 0 {
		return []string{"Inbox"}
	}
	return folders
}

func parseCommaList(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func (c *Config) validateScoringWeights() error {
	weights := []struct {
		name  string
		value float64
	}{
		{"skill", c.Scoring.Weights.Skill},
		{"experience", c.Scoring.Weights.Experience},
		{"location", c.Scoring.Weights.Location},
		{"salary", c.Scoring.Weights.Salary},
		{"description", c.Scoring.Weights.Description},
	}
	for _, w := range weights {
		if w.value < 0 {
			return fmt.Errorf("config: scoring weight %s must be non-negative, got %.2f", w.name, w.value)
		}
	}
	sum := c.Scoring.Weights.Skill + c.Scoring.Weights.Experience +
		c.Scoring.Weights.Location + c.Scoring.Weights.Salary +
		c.Scoring.Weights.Description
	if math.Abs(sum-1.0) > 0.01 {
		return fmt.Errorf("config: scoring weights must sum to 1.0, got %.2f", sum)
	}
	return nil
}
