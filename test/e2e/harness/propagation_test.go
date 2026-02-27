package harness

import (
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
	"github.com/stretchr/testify/assert"
)

func TestCheckStrictness_SkewThresholds(t *testing.T) {
	val2000 := int64(2000)
	val1000 := int64(1000)

	tests := []struct {
		name        string
		cfg         SessionConfig
		rel         summary.Reliability
		expectError bool
	}{
		{
			name:        "BestEffort_NoThresholds",
			cfg:         SessionConfig{StrictnessMode: "BestEffort"},
			rel:         summary.Reliability{StartSkewMs: &val2000},
			expectError: false,
		},
		{
			name:        "BestEffort_ExceedThresholds",
			cfg:         SessionConfig{StrictnessMode: "BestEffort", MaxStartSkewMs: 1000},
			rel:         summary.Reliability{StartSkewMs: &val2000},
			expectError: false, // BestEffort 모드에서는 무시되고 nil을 반환해야 함
		},
		{
			name:        "StrictCollection_UnderThresholds",
			cfg:         SessionConfig{StrictnessMode: "StrictCollection", MaxStartSkewMs: 3000},
			rel:         summary.Reliability{StartSkewMs: &val2000, CollectionStatus: "Complete"},
			expectError: false,
		},
		{
			name:        "StrictCollection_ExceedStartSkew",
			cfg:         SessionConfig{StrictnessMode: "StrictCollection", MaxStartSkewMs: 1500},
			rel:         summary.Reliability{StartSkewMs: &val2000, CollectionStatus: "Complete"},
			expectError: true, // 임계값 초과
		},
		{
			name:        "StrictCollection_ExceedEndSkew",
			cfg:         SessionConfig{StrictnessMode: "StrictCollection", MaxEndSkewMs: 500},
			rel:         summary.Reliability{EndSkewMs: &val1000, CollectionStatus: "Complete"},
			expectError: true,
		},
		{
			name: "StrictEvaluation_ExceedScrapeLatency",
			cfg:  SessionConfig{StrictnessMode: "StrictEvaluation", MaxScrapeLatencyMs: 1500},
			rel: summary.Reliability{
				ScrapeLatencyMs:  &val2000,
				CollectionStatus: "Complete",
				EvaluationStatus: "Complete",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sum := &summary.Summary{
				Reliability: &tt.rel,
				Results:     []summary.SLIResult{},
			}
			err := CheckStrictness(tt.cfg, sum)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCheckGating(t *testing.T) {
	tests := []struct {
		name        string
		gateOnLevel string
		results     []summary.SLIResult
		expectError bool
		errorMsg    string
	}{
		{
			name:        "NoneMode_ShouldPass",
			gateOnLevel: "none",
			results: []summary.SLIResult{
				{ID: "sli1", Status: summary.StatusFail},
			},
			expectError: false,
		},
		{
			name:        "EmptyMode_ShouldPass",
			gateOnLevel: "",
			results: []summary.SLIResult{
				{ID: "sli1", Status: summary.StatusFail},
			},
			expectError: false,
		},
		{
			name:        "FailMode_WithPass_ShouldPass",
			gateOnLevel: "fail",
			results: []summary.SLIResult{
				{ID: "sli1", Status: summary.StatusPass},
				{ID: "sli2", Status: summary.StatusWarn}, // fail 모드에서 warn은 무시됨
			},
			expectError: false,
		},
		{
			name:        "FailMode_WithFail_ShouldFail",
			gateOnLevel: "fail",
			results: []summary.SLIResult{
				{ID: "sli1", Status: summary.StatusPass},
				{ID: "sli2", Status: summary.StatusFail, Reason: "threshold crossed"},
			},
			expectError: true,
			errorMsg:    "GatingPolicy violation (fail): SLI sli2 failed: threshold crossed",
		},
		{
			name:        "WarnMode_WithWarn_ShouldFail",
			gateOnLevel: "warn",
			results: []summary.SLIResult{
				{ID: "sli1", Status: summary.StatusPass},
				{ID: "sli2", Status: summary.StatusWarn, Reason: "nearing limit"},
			},
			expectError: true,
			errorMsg:    "GatingPolicy violation (warn): SLI sli2 triggered level warn: nearing limit",
		},
		{
			name:        "WarnMode_WithFail_ShouldFail",
			gateOnLevel: "warn",
			results: []summary.SLIResult{
				{ID: "sli1", Status: summary.StatusFail, Reason: "dead"},
			},
			expectError: true,
			errorMsg:    "GatingPolicy violation (warn): SLI sli1 triggered level fail: dead",
		},
		{
			name:        "SkipAndBlock_ShouldBeIgnored",
			gateOnLevel: "warn",
			results: []summary.SLIResult{
				{ID: "sli1", Status: summary.StatusSkip, Reason: "missing inputs"},
				{ID: "sli2", Status: summary.StatusBlock, Reason: "blocked by strictness"},
			},
			expectError: false,
		},
		{
			name:        "MultipleFailures_ShouldCombineReasons",
			gateOnLevel: "fail",
			results: []summary.SLIResult{
				{ID: "sli1", Status: summary.StatusFail, Reason: "r1"},
				{ID: "sli2", Status: summary.StatusFail, Reason: "r2"},
			},
			expectError: true,
			errorMsg:    "GatingPolicy violation (fail): SLI sli1 failed: r1; SLI sli2 failed: r2",
		},
		{
			name:        "NilSummary_ShouldFail",
			gateOnLevel: "fail",
			results:     nil, // 의도적으로 nil 전달을 위해 아래 t.Run에서 핸들링
			expectError: true,
			errorMsg:    "summary is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sum *summary.Summary
			if tt.name != "NilSummary_ShouldFail" {
				sum = &summary.Summary{
					Results: tt.results,
				}
			}

			cfg := SessionConfig{GateOnLevel: tt.gateOnLevel}
			err := CheckGating(cfg, sum)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Equal(t, tt.errorMsg, err.Error())
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
