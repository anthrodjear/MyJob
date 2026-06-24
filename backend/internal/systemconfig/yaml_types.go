// Package systemconfig provides configuration resolution logic for the job search agent.
// This file defines the YAML structure types used to parse config/application.yaml.
//
// # Design Constraints
//
//   - These types mirror the YAML structure exactly — they are NOT the API response types.
//   - EffectiveConfig (in model.go) is the API response type; these types are internal.
//   - Prompts are intentionally excluded — they stay in code and are never exposed.
package systemconfig

// ---------------------------------------------------------------------------
// YAMLConfig — mirrors config/application.yaml structure
// ---------------------------------------------------------------------------

// YAMLConfig mirrors the structure of config/application.yaml for parsing.
// Fields use yaml tags matching the YAML keys exactly. This struct captures
// only the sections needed for EffectiveConfig — prompts are intentionally excluded.
type YAMLConfig struct {
	Application YAMLApplication `yaml:"application"`
	Queue       YAMLQueue       `yaml:"queue"`
	LLM         YAMLLLM         `yaml:"llm"`
	Voice       YAMLVoice       `yaml:"voice"`
	Interview   YAMLInterview   `yaml:"interview"`
	Email       YAMLEmail       `yaml:"email"`
}

// YAMLApplication holds application-level settings from YAML.
type YAMLApplication struct {
	ApprovalTiers YAMLApprovalTiers `yaml:"approval_tiers"`
	AutoGenerate  YAMLAutoGenerate  `yaml:"auto_generate"`
	Resume        YAMLResume        `yaml:"resume"`
	CoverLetter   YAMLCoverLetter   `yaml:"cover_letter"`
}

// YAMLApprovalTiers holds auto/review/reject tier definitions.
type YAMLApprovalTiers struct {
	AutoApply YAMLTierDef `yaml:"auto_apply"`
	Review    YAMLTierDef `yaml:"review"`
	Reject    YAMLTierDef `yaml:"reject"`
}

// YAMLTierDef holds a single tier definition from YAML.
type YAMLTierDef struct {
	MinScore int    `yaml:"min_score"`
	MaxScore int    `yaml:"max_score"`
	Action   string `yaml:"action"`
	Notify   bool   `yaml:"notify"`
	Log      bool   `yaml:"log"`
}

// YAMLAutoGenerate holds auto-generation toggles.
type YAMLAutoGenerate struct {
	Resume      bool `yaml:"resume"`
	CoverLetter bool `yaml:"cover_letter"`
}

// YAMLResume holds resume generation settings.
type YAMLResume struct {
	Engine      string `yaml:"engine"`
	TemplateDir string `yaml:"template_dir"`
}

// YAMLCoverLetter holds cover letter generation settings.
type YAMLCoverLetter struct {
	Engine      string `yaml:"engine"`
	TemplateDir string `yaml:"template_dir"`
	MaxLength   int    `yaml:"max_length"`
}

// YAMLQueue holds queue settings from YAML.
type YAMLQueue struct {
	Concurrency   int `yaml:"concurrency"`
	RetryAttempts int `yaml:"retryAttempts"`
}

// YAMLLLM holds LLM provider settings from YAML.
type YAMLLLM struct {
	Primary    YAMLLLMProvider `yaml:"primary"`
	Local      YAMLLLMProvider `yaml:"local"`
	Embeddings YAMLLLMProvider `yaml:"embeddings"`
}

// YAMLLLMProvider holds a single LLM provider's YAML config.
type YAMLLLMProvider struct {
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
}

// YAMLVoice holds voice provider settings from YAML.
type YAMLVoice struct {
	Provider string           `yaml:"provider"`
	Model    string           `yaml:"model"`
	LiveKit  YAMLLiveKit      `yaml:"livekit"`
}

// YAMLLiveKit holds LiveKit connection settings.
type YAMLLiveKit struct {
	URL       string `yaml:"url"`
	APIKey    string `yaml:"api_key"`
	APISecret string `yaml:"api_secret"`
}

// YAMLInterview holds interview agent settings from YAML.
type YAMLInterview struct {
	Memory    YAMLInterviewMemory    `yaml:"memory"`
	Responder YAMLInterviewResponder `yaml:"responder"`
	Planner   YAMLInterviewPlanner   `yaml:"planner"`
}

// YAMLInterviewMemory holds transcript window settings.
type YAMLInterviewMemory struct {
	MaxRecentSegments  int `yaml:"max_recent_segments"`
	KeepAfterSummarize int `yaml:"keep_after_summarize"`
}

// YAMLInterviewResponder holds LLM settings for the responder.
type YAMLInterviewResponder struct {
	LLM YAMLLLMTimeout `yaml:"llm"`
}

// YAMLLLMTimeout holds timeout settings.
type YAMLLLMTimeout struct {
	TimeoutMs int `yaml:"timeout_ms"`
	Retries   int `yaml:"retries"`
}

// YAMLInterviewPlanner holds decision thresholds.
type YAMLInterviewPlanner struct {
	DuplicateThreshold  float64 `yaml:"duplicate_threshold"`
	MinSubstantiveLength int    `yaml:"min_substantive_length"`
}

// YAMLEmail holds email settings from YAML.
type YAMLEmail struct {
	Provider      string   `yaml:"provider"`
	CheckInterval string   `yaml:"check_interval"`
	Folders       []string `yaml:"folders"`
}
