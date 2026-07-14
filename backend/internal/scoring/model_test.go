package scoring

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApprovalTierConstants(t *testing.T) {
	assert.Equal(t, ApprovalTier("reject"), TierReject)
	assert.Equal(t, ApprovalTier("review"), TierReview)
	assert.Equal(t, ApprovalTier("auto"), TierAuto)
	assert.Equal(t, float64(50), NeutralScore)
}

func TestTier(t *testing.T) {
	tests := []struct {
		name            string
		score           float64
		autoThreshold   int
		reviewThreshold int
		want            ApprovalTier
	}{
		// auto: score >= autoThreshold
		{"auto exact threshold", 95, 95, 80, TierAuto},
		{"auto above threshold", 97.5, 95, 80, TierAuto},
		{"auto high score", 100, 95, 80, TierAuto},
		{"auto with low thresholds", 80, 80, 60, TierAuto},
		// review: score >= reviewThreshold but < autoThreshold
		{"review exact threshold", 80, 95, 80, TierReview},
		{"review mid range", 85, 95, 80, TierReview},
		{"review just below auto", 94.9, 95, 80, TierReview},
		// reject: score < reviewThreshold
		{"reject zero", 0, 95, 80, TierReject},
		{"reject just below review", 79.9, 95, 80, TierReject},
		{"reject low score", 30, 95, 80, TierReject},
		// boundary: score = 0
		{"reject score zero", 0, 90, 70, TierReject},
		// thresholds equal
		{"auto when thresholds equal", 90, 90, 90, TierAuto},
		{"review when thresholds equal", 89, 90, 90, TierReject},
		// negative score
		{"reject negative score", -5, 95, 80, TierReject},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Tier(tt.score, tt.autoThreshold, tt.reviewThreshold)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestScoreResult_Fields(t *testing.T) {
	details := &ScoreDetails{
		SkillMatch:       90,
		ExperienceMatch:  85,
		LocationMatch:    70,
		SalaryMatch:      80,
		DescriptionMatch: 75,
	}

	r := ScoreResult{
		Score:     87.5,
		Tier:      TierReview,
		Reasoning: "Strong skill match but location not ideal",
		Source:    "hybrid",
		Model:     "gpt-4o",
		Details:   details,
	}

	assert.Equal(t, 87.5, r.Score)
	assert.Equal(t, TierReview, r.Tier)
	assert.Equal(t, "Strong skill match but location not ideal", r.Reasoning)
	assert.Equal(t, "hybrid", r.Source)
	assert.Equal(t, "gpt-4o", r.Model)
	assert.NotNil(t, r.Details)
	assert.Equal(t, 90.0, r.Details.SkillMatch)
	assert.Equal(t, 85.0, r.Details.ExperienceMatch)
	assert.Equal(t, 70.0, r.Details.LocationMatch)
	assert.Equal(t, 80.0, r.Details.SalaryMatch)
	assert.Equal(t, 75.0, r.Details.DescriptionMatch)
}

func TestScoreResult_NilDetails(t *testing.T) {
	r := ScoreResult{
		Score:  95,
		Tier:   TierAuto,
		Source: "heuristic",
	}
	assert.Equal(t, 95.0, r.Score)
	assert.Equal(t, TierAuto, r.Tier)
	assert.Nil(t, r.Details)
	assert.Empty(t, r.Reasoning)
	assert.Empty(t, r.Model)
}

func TestScoreResult_JSONRoundTrip(t *testing.T) {
	details := &ScoreDetails{
		SkillMatch:       90,
		ExperienceMatch:  85,
		LocationMatch:    70,
		SalaryMatch:      80,
		DescriptionMatch: 75,
	}

	r := ScoreResult{
		Score:     87.5,
		Tier:      TierReview,
		Reasoning: "Strong skill match",
		Source:    "hybrid",
		Model:     "gpt-4o",
		Details:   details,
	}

	data, err := json.Marshal(r)
	require.NoError(t, err)

	var decoded ScoreResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, r.Score, decoded.Score)
	assert.Equal(t, r.Tier, decoded.Tier)
	assert.Equal(t, r.Reasoning, decoded.Reasoning)
	assert.Equal(t, r.Source, decoded.Source)
	assert.Equal(t, r.Model, decoded.Model)
	require.NotNil(t, decoded.Details)
	assert.Equal(t, r.Details.SkillMatch, decoded.Details.SkillMatch)
}

func TestScoreResult_JSONNilDetails(t *testing.T) {
	r := ScoreResult{
		Score:  95,
		Tier:   TierAuto,
		Source: "heuristic",
	}

	data, err := json.Marshal(r)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"score":95`)
	assert.Contains(t, string(data), `"tier":"auto"`)

	var decoded ScoreResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Nil(t, decoded.Details)
}

func TestScoreDetails_Fields(t *testing.T) {
	d := ScoreDetails{
		SkillMatch:       92.5,
		ExperienceMatch:  88.0,
		LocationMatch:    100.0,
		SalaryMatch:      75.0,
		DescriptionMatch: 0.0,
	}

	assert.Equal(t, 92.5, d.SkillMatch)
	assert.Equal(t, 88.0, d.ExperienceMatch)
	assert.Equal(t, 100.0, d.LocationMatch)
	assert.Equal(t, 75.0, d.SalaryMatch)
	assert.Equal(t, 0.0, d.DescriptionMatch)
}

func TestScoreDetails_Validate(t *testing.T) {
	t.Run("valid all factors in range", func(t *testing.T) {
		d := ScoreDetails{
			SkillMatch:       100,
			ExperienceMatch:  85.5,
			LocationMatch:    70,
			SalaryMatch:      0,
			DescriptionMatch: 50,
		}
		assert.NoError(t, d.Validate())
	})

	t.Run("valid zero values", func(t *testing.T) {
		d := ScoreDetails{}
		assert.NoError(t, d.Validate())
	})

	t.Run("valid boundary 100", func(t *testing.T) {
		d := ScoreDetails{
			SkillMatch:       100,
			ExperienceMatch:  100,
			LocationMatch:    100,
			SalaryMatch:      100,
			DescriptionMatch: 100,
		}
		assert.NoError(t, d.Validate())
	})

	t.Run("invalid skill_match below 0", func(t *testing.T) {
		d := ScoreDetails{SkillMatch: -1}
		err := d.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "skill_match")
	})

	t.Run("invalid skill_match above 100", func(t *testing.T) {
		d := ScoreDetails{SkillMatch: 101}
		err := d.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "skill_match")
	})

	t.Run("invalid experience_match", func(t *testing.T) {
		d := ScoreDetails{ExperienceMatch: -0.1}
		err := d.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "experience_match")
	})

	t.Run("invalid location_match", func(t *testing.T) {
		d := ScoreDetails{LocationMatch: 150}
		err := d.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "location_match")
	})

	t.Run("invalid salary_match", func(t *testing.T) {
		d := ScoreDetails{SalaryMatch: -5}
		err := d.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "salary_match")
	})

	t.Run("invalid description_match", func(t *testing.T) {
		d := ScoreDetails{DescriptionMatch: 999}
		err := d.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "description_match")
	})
}

func TestWeights_Fields(t *testing.T) {
	w := Weights{
		Skill:       0.4,
		Experience:  0.3,
		Location:    0.1,
		Salary:      0.1,
		Description: 0.1,
	}

	assert.Equal(t, 0.4, w.Skill)
	assert.Equal(t, 0.3, w.Experience)
	assert.Equal(t, 0.1, w.Location)
	assert.Equal(t, 0.1, w.Salary)
	assert.Equal(t, 0.1, w.Description)
}

func TestWeights_Validate(t *testing.T) {
	t.Run("valid weights sum to 1", func(t *testing.T) {
		w := Weights{
			Skill:       0.4,
			Experience:  0.3,
			Location:    0.1,
			Salary:      0.1,
			Description: 0.1,
		}
		assert.NoError(t, w.Validate())
	})

	t.Run("valid weights non-normalized", func(t *testing.T) {
		w := Weights{
			Skill:       2,
			Experience:  1,
			Location:    1,
			Salary:      0.5,
			Description: 0.5,
		}
		assert.NoError(t, w.Validate())
	})

	t.Run("valid single weight", func(t *testing.T) {
		w := Weights{Skill: 1}
		assert.NoError(t, w.Validate())
	})

	t.Run("valid decimals", func(t *testing.T) {
		w := Weights{
			Skill:       0.25,
			Experience:  0.25,
			Location:    0.25,
			Salary:      0.25,
			Description: 0,
		}
		assert.NoError(t, w.Validate())
	})

	t.Run("invalid negative skill weight", func(t *testing.T) {
		w := Weights{Skill: -0.1}
		err := w.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "weight skill")
	})

	t.Run("invalid negative experience weight", func(t *testing.T) {
		w := Weights{Experience: -0.5}
		err := w.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "weight experience")
	})

	t.Run("invalid negative location weight", func(t *testing.T) {
		w := Weights{Location: -1}
		err := w.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "weight location")
	})

	t.Run("invalid negative salary weight", func(t *testing.T) {
		w := Weights{Salary: -0.01}
		err := w.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "weight salary")
	})

	t.Run("invalid negative description weight", func(t *testing.T) {
		w := Weights{Description: -100}
		err := w.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "weight description")
	})

	t.Run("invalid all zero weights", func(t *testing.T) {
		w := Weights{}
		err := w.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must not all be zero")
	})
}

func TestNormalizeScore(t *testing.T) {
	tests := []struct {
		name  string
		input float64
		want  float64
	}{
		{"within range", 75, 75},
		{"zero", 0, 0},
		{"exactly 100", 100, 100},
		{"negative clamped to 0", -10, 0},
		{"negative small", -0.01, 0},
		{"above 100 clamped", 150, 100},
		{"above 100 small", 100.01, 100},
		{"exactly 50", 50, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NormalizeScore(tt.input))
		})
	}
}

func TestComputeScore(t *testing.T) {
	tests := []struct {
		name    string
		details ScoreDetails
		weights Weights
		want    float64
	}{
		{
			name: "perfect score",
			details: ScoreDetails{
				SkillMatch:       100,
				ExperienceMatch:  100,
				LocationMatch:    100,
				SalaryMatch:      100,
				DescriptionMatch: 100,
			},
			weights: Weights{
				Skill:       0.4,
				Experience:  0.3,
				Location:    0.1,
				Salary:      0.1,
				Description: 0.1,
			},
			want: 100,
		},
		{
			name: "zero details weighted",
			details: ScoreDetails{
				SkillMatch:       0,
				ExperienceMatch:  0,
				LocationMatch:    0,
				SalaryMatch:      0,
				DescriptionMatch: 0,
			},
			weights: Weights{
				Skill:       0.4,
				Experience:  0.3,
				Location:    0.1,
				Salary:      0.1,
				Description: 0.1,
			},
			want: 0,
		},
		{
			name: "mixed scores",
			details: ScoreDetails{
				SkillMatch:       100,
				ExperienceMatch:  80,
				LocationMatch:    50,
				SalaryMatch:      0,
				DescriptionMatch: 75,
			},
			weights: Weights{
				Skill:       0.4,
				Experience:  0.2,
				Location:    0.2,
				Salary:      0.1,
				Description: 0.1,
			},
			// (100*0.4 + 80*0.2 + 50*0.2 + 0*0.1 + 75*0.1) / (0.4+0.2+0.2+0.1+0.1)
			// = (40 + 16 + 10 + 0 + 7.5) / 1.0 = 73.5
			want: 73.5,
		},
		{
			name: "single weight only skill",
			details: ScoreDetails{
				SkillMatch:       90,
				ExperienceMatch:  50,
				LocationMatch:    20,
				SalaryMatch:      10,
				DescriptionMatch: 0,
			},
			weights: Weights{Skill: 1},
			want:    90,
		},
		{
			name: "non-normalized weights",
			details: ScoreDetails{
				SkillMatch:       100,
				ExperienceMatch:  50,
				LocationMatch:    0,
				SalaryMatch:      0,
				DescriptionMatch: 0,
			},
			weights: Weights{
				Skill:      2,
				Experience: 1,
			},
			// (100*2 + 50*1 + 0 + 0 + 0) / (2+1) = 250/3 = 83.33... → 83.33
			want: 83.33,
		},
		{
			name: "zero total weight",
			details: ScoreDetails{
				SkillMatch: 100,
			},
			weights: Weights{},
			want:    0,
		},
		{
			name: "rounding to 2 decimal places",
			details: ScoreDetails{
				SkillMatch:       100,
				ExperienceMatch:  100,
				LocationMatch:    100,
				SalaryMatch:      100,
				DescriptionMatch: 33,
			},
			weights: Weights{
				Skill:       1,
				Experience:  1,
				Location:    1,
				Salary:      1,
				Description: 1,
			},
			// (100+100+100+100+33) / 5 = 433/5 = 86.6
			want: 86.6,
		},
		{
			name: "clamp negative result",
			details: ScoreDetails{
				SkillMatch:       -10,
				ExperienceMatch:  0,
				LocationMatch:    0,
				SalaryMatch:      0,
				DescriptionMatch: 0,
			},
			weights: Weights{Skill: 1},
			want:    0,
		},
		{
			name: "clamp above 100",
			details: ScoreDetails{
				SkillMatch:       200,
				ExperienceMatch:  0,
				LocationMatch:    0,
				SalaryMatch:      0,
				DescriptionMatch: 0,
			},
			weights: Weights{Skill: 1},
			want:    100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeScore(tt.details, tt.weights)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeSkills(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "mixed case and spaces",
			input: []string{"  Go ", "Python", "  REACT  ", "SQL"},
			want:  []string{"go", "python", "react", "sql"},
		},
		{
			name:  "already normalized",
			input: []string{"go", "python", "react"},
			want:  []string{"go", "python", "react"},
		},
		{
			name:  "empty list",
			input: []string{},
			want:  []string{},
		},
		{
			name:  "single element",
			input: []string{"  KubernETEs  "},
			want:  []string{"kubernetes"},
		},
		{
			name:  "trailing only spaces",
			input: []string{"aws "},
			want:  []string{"aws"},
		},
		{
			name:  "preserves order",
			input: []string{"A", " B", "C "},
			want:  []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeSkills(tt.input)
			assert.Equal(t, tt.want, got)
			// Verify original slice is not mutated
			if len(tt.input) > 0 {
				assert.NotSame(t, &tt.input[0], &got[0], "should return a new slice")
			}
		})
	}
}

func TestNormalizeSkills_NilInput(t *testing.T) {
	got := NormalizeSkills(nil)
	assert.Empty(t, got)
	assert.NotNil(t, got) // make returns empty slice, not nil
}

func TestSkillOverlap(t *testing.T) {
	tests := []struct {
		name      string
		required  []string
		candidate []string
		want      float64
	}{
		{
			name:      "perfect match",
			required:  []string{"go", "python", "sql"},
			candidate: []string{"go", "python", "sql"},
			want:      100,
		},
		{
			name:      "partial match",
			required:  []string{"go", "python", "sql", "docker"},
			candidate: []string{"go", "python"},
			want:      50,
		},
		{
			name:      "no match",
			required:  []string{"go", "python"},
			candidate: []string{"java", "c++"},
			want:      0,
		},
		{
			name:      "empty required returns 100",
			required:  []string{},
			candidate: []string{"anything"},
			want:      100,
		},
		{
			name:      "empty candidate",
			required:  []string{"go", "python"},
			candidate: []string{},
			want:      0,
		},
		{
			name:      "duplicates in required are deduplicated",
			required:  []string{"go", "go", "python", "python"},
			candidate: []string{"go"},
			want:      50,
		},
		{
			name:      "extra candidate skills don't affect score",
			required:  []string{"go"},
			candidate: []string{"go", "python", "sql", "docker"},
			want:      100,
		},
		{
			name:      "both empty",
			required:  []string{},
			candidate: []string{},
			want:      100,
		},
		{
			name:      "case sensitive (must be normalized by caller)",
			required:  []string{"Go"},
			candidate: []string{"go"},
			want:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SkillOverlap(tt.required, tt.candidate)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSkillOverlap_NilInputs(t *testing.T) {
	t.Run("nil required", func(t *testing.T) {
		got := SkillOverlap(nil, []string{"go"})
		assert.Equal(t, float64(100), got)
	})

	t.Run("nil candidate", func(t *testing.T) {
		got := SkillOverlap([]string{"go", "python"}, nil)
		assert.Equal(t, float64(0), got)
	})

	t.Run("both nil", func(t *testing.T) {
		got := SkillOverlap(nil, nil)
		assert.Equal(t, float64(100), got)
	})
}

func TestKeywordOverlap(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		keywords []string
		want     float64
	}{
		{
			name:     "all keywords match",
			text:     "We are looking for a Go developer with SQL experience",
			keywords: []string{"go", "sql", "developer"},
			want:     100,
		},
		{
			name:     "partial match",
			text:     "We need a Go developer",
			keywords: []string{"go", "python", "react"},
			want:     float64(1) / float64(3) * 100, // 33.33...
		},
		{
			name:     "no match",
			text:     "We need a Java developer",
			keywords: []string{"go", "python"},
			want:     0,
		},
		{
			name:     "empty keywords returns NeutralScore",
			text:     "Any text here",
			keywords: []string{},
			want:     NeutralScore,
		},
		{
			name:     "case insensitive matching",
			text:     "Golang Developer Needed",
			keywords: []string{"golang", "developer"},
			want:     100,
		},
		{
			name:     "substring matching - sql matches postgresql",
			text:     "I know postgresql and aws",
			keywords: []string{"sql", "aws"},
			want:     100, // both match: "sql" is substring of "postgresql", "aws" matches directly
		},
		{
			name:     "partial substring match",
			text:     "I know postgresql only",
			keywords: []string{"sql", "aws"},
			want:     50, // only "sql" matches as substring
		},
		{
			name:     "empty text",
			text:     "",
			keywords: []string{"go", "python"},
			want:     0,
		},
		{
			name:     "both empty",
			text:     "",
			keywords: []string{},
			want:     NeutralScore,
		},
		{
			name:     "single keyword match",
			text:     "Expert in machine learning",
			keywords: []string{"machine learning"},
			want:     100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := KeywordOverlap(tt.text, tt.keywords)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestKeywordOverlap_NilKeywords(t *testing.T) {
	got := KeywordOverlap("some job description", nil)
	assert.Equal(t, NeutralScore, got)
}
