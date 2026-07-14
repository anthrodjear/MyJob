package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// yamlPromptsConfig maps the application.yaml structure for prompts.
// prompts: is a root-level key in the YAML, not nested under application:.
type yamlPromptsConfig struct {
	Prompts yamlPrompts `yaml:"prompts"`
}

// yamlPrompts maps YAML keys to PromptPair fields.
type yamlPrompts struct {
	Scoring           yamlPromptPair `yaml:"scoring"`
	EmailClassifier   yamlPromptPair `yaml:"email_classifier"`
	CoverLetter       yamlPromptPair `yaml:"cover_letter"`
	ResumeTailor      yamlPromptPair `yaml:"resume_tailor"`
	InterviewPrep     yamlPromptPair `yaml:"interview_prep"`
	JobExtraction     yamlPromptPair `yaml:"job_extraction"`
	FormUnderstanding yamlPromptPair `yaml:"form_understanding"`
	ResumeGeneration  yamlPromptPair `yaml:"resume_generation"`
}

type yamlPromptPair struct {
	System string `yaml:"system"`
	User   string `yaml:"user"`
}

// LoadPromptsFromYAML parses prompts from raw application.yaml bytes.
// Returns empty PromptsConfig if data is nil or parsing fails.
func LoadPromptsFromYAML(data []byte) PromptsConfig {
	if len(data) == 0 {
		return PromptsConfig{}
	}

	var yamlCfg yamlPromptsConfig
	if err := yaml.Unmarshal(data, &yamlCfg); err != nil {
		// Logger not available at config load time — fmt.Printf is intentional here.
		// This only fires on YAML parse failure during startup.
		fmt.Printf("config: failed to parse application.yaml prompts: %v\n", err)
		return PromptsConfig{}
	}

	p := yamlCfg.Prompts
	return PromptsConfig{
		Scoring:           toPromptPair(p.Scoring),
		EmailClassifier:   toPromptPair(p.EmailClassifier),
		CoverLetter:       toPromptPair(p.CoverLetter),
		ResumeTailor:      toPromptPair(p.ResumeTailor),
		InterviewPrep:     toPromptPair(p.InterviewPrep),
		JobExtraction:     toPromptPair(p.JobExtraction),
		FormUnderstanding: toPromptPair(p.FormUnderstanding),
		ResumeGeneration:  toPromptPair(p.ResumeGeneration),
	}
}

func toPromptPair(yp yamlPromptPair) PromptPair {
	return PromptPair(yp)
}
