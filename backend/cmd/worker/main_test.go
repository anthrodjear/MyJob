package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestParseRedisAddr(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "plain host:port",
			url:  "localhost:6379",
			want: "localhost:6379",
		},
		{
			name: "redis:// URL",
			url:  "redis://localhost:6379",
			want: "localhost:6379",
		},
		{
			name: "redis:// with password",
			url:  "redis://:secret@localhost:6379",
			want: "localhost:6379",
		},
		{
			name: "redis:// without port",
			url:  "redis://localhost",
			want: "localhost:6379",
		},
		{
			name: "rediss:// TLS URL",
			url:  "rediss://my-redis.example.com:6380",
			want: "my-redis.example.com:6380",
		},
		{
			name: "redis:// with path",
			url:  "redis://localhost:6379/0",
			want: "localhost:6379",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseRedisAddr(tt.url, logger)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		want         string
	}{
		{
			name:         "env var set",
			key:          "TEST_GET_ENV_SET",
			defaultValue: "default",
			envValue:     "custom",
			want:         "custom",
		},
		{
			name:         "env var not set returns default",
			key:          "TEST_GET_ENV_UNSET_12345",
			defaultValue: "default",
			envValue:     "",
			want:         "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			}
			got := getEnv(tt.key, tt.defaultValue)
			assert.Equal(t, tt.want, got)
		})
	}
}
