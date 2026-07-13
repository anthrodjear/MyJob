package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"backend/internal/config"
	"backend/internal/emails"
)

func TestToEmailPromptPair(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.PromptPair
		want emails.PromptPair
	}{
		{
			name: "basic conversion",
			cfg: config.PromptPair{
				System: "You are an email classifier",
				User:   "Classify this email",
			},
			want: emails.PromptPair{
				System: "You are an email classifier",
				User:   "Classify this email",
			},
		},
		{
			name: "empty values",
			cfg:  config.PromptPair{},
			want: emails.PromptPair{},
		},
		{
			name: "special characters",
			cfg: config.PromptPair{
				System: "System prompt with \"quotes\" and \\backslashes",
				User:   "User prompt with newlines\nand tabs",
			},
			want: emails.PromptPair{
				System: "System prompt with \"quotes\" and \\backslashes",
				User:   "User prompt with newlines\nand tabs",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toEmailPromptPair(tt.cfg)
			assert.Equal(t, tt.want, got)
		})
	}
}
