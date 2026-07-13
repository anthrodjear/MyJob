package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadPromptsFromYAML(t *testing.T) {
	tests := []struct {
		name     string
		yamlData []byte
		expected PromptsConfig
	}{
		{
			name: "valid complete YAML",
			yamlData: []byte(`
prompts:
  scoring:
    system: "You are a scoring assistant."
    user: "Score this job: {{.job}}"
  email_classifier:
    system: "Classify emails."
    user: "Classify: {{.email}}"
  cover_letter:
    system: "Write cover letters."
    user: "Write for {{.job}}"
  resume_tailor:
    system: "Tailor resumes."
    user: "Tailor for {{.job}}"
  interview_prep:
    system: "Prepare interviews."
    user: "Prepare for {{.job}}"
  job_extraction:
    system: "Extract job data."
    user: "Extract from {{.text}}"
  form_understanding:
    system: "Understand forms."
    user: "Understand {{.form}}"
  resume_generation:
    system: "Generate resumes."
    user: "Generate for {{.profile}}"
`),
			expected: PromptsConfig{
				Scoring: PromptPair{
					System: "You are a scoring assistant.",
					User:   "Score this job: {{.job}}",
				},
				EmailClassifier: PromptPair{
					System: "Classify emails.",
					User:   "Classify: {{.email}}",
				},
				CoverLetter: PromptPair{
					System: "Write cover letters.",
					User:   "Write for {{.job}}",
				},
				ResumeTailor: PromptPair{
					System: "Tailor resumes.",
					User:   "Tailor for {{.job}}",
				},
				InterviewPrep: PromptPair{
					System: "Prepare interviews.",
					User:   "Prepare for {{.job}}",
				},
				JobExtraction: PromptPair{
					System: "Extract job data.",
					User:   "Extract from {{.text}}",
				},
				FormUnderstanding: PromptPair{
					System: "Understand forms.",
					User:   "Understand {{.form}}",
				},
				ResumeGeneration: PromptPair{
					System: "Generate resumes.",
					User:   "Generate for {{.profile}}",
				},
			},
		},
		{
			name:     "empty data returns empty config",
			yamlData: []byte{},
			expected: PromptsConfig{},
		},
		{
			name: "nil data returns empty config",
			yamlData: nil,
			expected: PromptsConfig{},
		},
		{
			name: "invalid YAML returns empty config",
			yamlData: []byte(`
prompts:
  scoring:
    system: "valid
    user: "also valid
`),
			expected: PromptsConfig{},
		},
		{
			name: "partial YAML - missing optional fields",
			yamlData: []byte(`
prompts:
  scoring:
    system: "Score this"
    user: "Rate this"
`),
			expected: PromptsConfig{
				Scoring: PromptPair{
					System: "Score this",
					User:   "Rate this",
				},
			},
		},
		{
			name: "YAML with empty strings",
			yamlData: []byte(`
prompts:
  scoring:
    system: ""
    user: ""
  email_classifier:
    system: ""
    user: ""
`),
			expected: PromptsConfig{
				Scoring: PromptPair{
					System: "",
					User:   "",
				},
				EmailClassifier: PromptPair{
					System: "",
					User:   "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LoadPromptsFromYAML(tt.yamlData)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToPromptPair(t *testing.T) {
	tests := []struct {
		name  string
		input yamlPromptPair
		expected PromptPair
	}{
		{
			name: "converts yaml to prompt pair",
			input: yamlPromptPair{
				System: "System prompt",
				User:   "User prompt",
			},
			expected: PromptPair{
				System: "System prompt",
				User:   "User prompt",
			},
		},
		{
			name: "handles empty strings",
			input: yamlPromptPair{
				System: "",
				User:   "",
			},
			expected: PromptPair{
				System: "",
				User:   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toPromptPair(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPromptsConfig_Equality(t *testing.T) {
	cfg1 := PromptsConfig{
		Scoring: PromptPair{
			System: "System 1",
			User:   "User 1",
		},
	}
	cfg2 := PromptsConfig{
		Scoring: PromptPair{
			System: "System 1",
			User:   "User 1",
		},
	}
	cfg3 := PromptsConfig{
		Scoring: PromptPair{
			System: "System 2",
			User:   "User 2",
		},
	}

	assert.Equal(t, cfg1, cfg2)
	assert.NotEqual(t, cfg1, cfg3)
}

func TestPromptsConfig_AllFields(t *testing.T) {
	cfg := PromptsConfig{
		Scoring:           PromptPair{System: "s1", User: "u1"},
		EmailClassifier:   PromptPair{System: "s2", User: "u2"},
		CoverLetter:       PromptPair{System: "s3", User: "u3"},
		ResumeTailor:      PromptPair{System: "s4", User: "u4"},
		InterviewPrep:     PromptPair{System: "s5", User: "u5"},
		JobExtraction:     PromptPair{System: "s6", User: "u6"},
		FormUnderstanding: PromptPair{System: "s7", User: "u7"},
		ResumeGeneration:  PromptPair{System: "s8", User: "u8"},
	}

	require.Equal(t, "s1", cfg.Scoring.System)
	require.Equal(t, "u1", cfg.Scoring.User)
	require.Equal(t, "s2", cfg.EmailClassifier.System)
	require.Equal(t, "u2", cfg.EmailClassifier.User)
	require.Equal(t, "s3", cfg.CoverLetter.System)
	require.Equal(t, "u3", cfg.CoverLetter.User)
	require.Equal(t, "s4", cfg.ResumeTailor.System)
	require.Equal(t, "u4", cfg.ResumeTailor.User)
	require.Equal(t, "s5", cfg.InterviewPrep.System)
	require.Equal(t, "u5", cfg.InterviewPrep.User)
	require.Equal(t, "s6", cfg.JobExtraction.System)
	require.Equal(t, "u6", cfg.JobExtraction.User)
	require.Equal(t, "s7", cfg.FormUnderstanding.System)
	require.Equal(t, "u7", cfg.FormUnderstanding.User)
	require.Equal(t, "s8", cfg.ResumeGeneration.System)
	require.Equal(t, "u8", cfg.ResumeGeneration.User)
}