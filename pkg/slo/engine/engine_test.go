package engine

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/slo/fetch"
	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
	"github.com/stretchr/testify/assert"
)

type mockWriter struct {
	lastWritten *summary.Summary
}

func (m *mockWriter) Write(path string, sum summary.Summary) error {
	m.lastWritten = &sum
	return nil
}

type mockDelayFetcher struct {
	startDelay time.Duration
	endDelay   time.Duration
}

func (m *mockDelayFetcher) Fetch(ctx context.Context, at time.Time) (fetch.Sample, error) {
	if at.Unix() == 1 { // StartedAt
		time.Sleep(m.startDelay)
	} else if at.Unix() == 2 { // FinishedAt
		time.Sleep(m.endDelay)
	}
	return fetch.Sample{Values: map[string]float64{}}, nil
}

func TestEngine_ScrapeLatencyMax(t *testing.T) {
	tests := []struct {
		name       string
		startDelay time.Duration
		endDelay   time.Duration
	}{
		{"StartSlower", 50 * time.Millisecond, 10 * time.Millisecond},
		{"EndSlower", 10 * time.Millisecond, 50 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher := &mockDelayFetcher{startDelay: tt.startDelay, endDelay: tt.endDelay}
			writer := &mockWriter{}
			eng := New(fetcher, writer, nil)

			req := ExecuteRequest{
				Config: RunConfig{
					StartedAt:  time.Unix(1, 0),
					FinishedAt: time.Unix(2, 0),
				},
				Reliability: &summary.Reliability{},
			}

			sum, err := eng.Execute(context.Background(), req)
			assert.NoError(t, err)

			maxDelay := tt.startDelay
			if tt.endDelay > maxDelay {
				maxDelay = tt.endDelay
			}

			// Milliseconds should be roughly >= maxDelay Ms
			assert.NotNil(t, sum.Reliability.ScrapeLatencyMs)
			assert.GreaterOrEqual(t, *sum.Reliability.ScrapeLatencyMs, maxDelay.Milliseconds())
		})
	}
}

func TestEngine_ConfidenceScore(t *testing.T) {
	val5001 := int64(5001)

	tests := []struct {
		name          string
		relIn         summary.Reliability
		expectedScore float64
	}{
		{
			name:          "Perfect",
			relIn:         summary.Reliability{CollectionStatus: "Complete"},
			expectedScore: 1.0,
		},
		{
			name:          "CollectionFailed",
			relIn:         summary.Reliability{CollectionStatus: "Failed"},
			expectedScore: 0.0,
		},
		{
			name:          "PartialEvalAndMissing",
			relIn:         summary.Reliability{CollectionStatus: "Complete", EvaluationStatus: "Partial", MissingInputs: []string{"A", "B"}},
			expectedScore: 0.6, // 1.0 - 0.2 - 0.2 = 0.6
		},
		{
			name:          "SkewsHigh",
			relIn:         summary.Reliability{CollectionStatus: "Complete", StartSkewMs: &val5001, ScrapeLatencyMs: &val5001},
			expectedScore: 0.8, // 1.0 - 0.1 - 0.1 = 0.8
		},
		{
			name:          "CappedPenalties",
			relIn:         summary.Reliability{CollectionStatus: "Complete", MissingInputs: []string{"A", "B", "C", "D"}, SkippedSLIs: []string{"X", "Y", "Z", "W"}},
			expectedScore: 0.4, // 1.0 - 0.3 (max missing) - 0.3 (max skipped) = 0.4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eng := &Engine{}
			eng.ensureConfidenceScore(&tt.relIn)
			assert.NotNil(t, tt.relIn.ConfidenceScore)
			// Using float math exact matching for these simple increments
			assert.InDelta(t, tt.expectedScore, *tt.relIn.ConfidenceScore, 0.001)
		})
	}
}

func TestSummary_JSON_ConfidenceScore(t *testing.T) {
	val := 0.85
	sum := summary.Summary{
		SchemaVersion: "slo.v3",
		Reliability: &summary.Reliability{
			ConfidenceScore: &val,
		},
	}
	
	b, err := json.Marshal(sum)
	assert.NoError(t, err)

	jsonStr := string(b)
	assert.Contains(t, jsonStr, `"schemaVersion":"slo.v3"`)
	assert.Contains(t, jsonStr, `"confidenceScore":0.85`)
}
