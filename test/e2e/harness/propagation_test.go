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
			name: "BestEffort_NoThresholds",
			cfg:  SessionConfig{StrictnessMode: "BestEffort"},
			rel:  summary.Reliability{StartSkewMs: &val2000},
			expectError: false,
		},
		{
			name: "BestEffort_ExceedThresholds",
			cfg:  SessionConfig{StrictnessMode: "BestEffort", MaxStartSkewMs: 1000},
			rel:  summary.Reliability{StartSkewMs: &val2000},
			expectError: false, // BestEffort 모드에서는 무시되고 nil을 반환해야 함
		},
		{
			name: "StrictCollection_UnderThresholds",
			cfg:  SessionConfig{StrictnessMode: "StrictCollection", MaxStartSkewMs: 3000},
			rel:  summary.Reliability{StartSkewMs: &val2000, CollectionStatus: "Complete"},
			expectError: false,
		},
		{
			name: "StrictCollection_ExceedStartSkew",
			cfg:  SessionConfig{StrictnessMode: "StrictCollection", MaxStartSkewMs: 1500},
			rel:  summary.Reliability{StartSkewMs: &val2000, CollectionStatus: "Complete"},
			expectError: true, // 임계값 초과
		},
		{
			name: "StrictCollection_ExceedEndSkew",
			cfg:  SessionConfig{StrictnessMode: "StrictCollection", MaxEndSkewMs: 500},
			rel:  summary.Reliability{EndSkewMs: &val1000, CollectionStatus: "Complete"},
			expectError: true,
		},
		{
			name: "StrictEvaluation_ExceedScrapeLatency",
			cfg:  SessionConfig{StrictnessMode: "StrictEvaluation", MaxScrapeLatencyMs: 1500},
			rel:  summary.Reliability{ScrapeLatencyMs: &val2000, CollectionStatus: "Complete", EvaluationStatus: "Complete"},
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
