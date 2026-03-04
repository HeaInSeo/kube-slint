package engine

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/slo/fetch"
	"github.com/HeaInSeo/kube-slint/pkg/slo/spec"
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
			name: "PartialEvalAndMissing",
			relIn: summary.Reliability{
				CollectionStatus: "Complete",
				EvaluationStatus: "Partial",
				MissingInputs:    []string{"A", "B"},
			},
			expectedScore: 0.6, // 1.0 - 0.2 - 0.2 = 0.6
		},
		{
			name:          "SkewsHigh",
			relIn:         summary.Reliability{CollectionStatus: "Complete", StartSkewMs: &val5001, ScrapeLatencyMs: &val5001},
			expectedScore: 0.8, // 1.0 - 0.1 - 0.1 = 0.8
		},
		{
			name: "CappedPenalties",
			relIn: summary.Reliability{
				CollectionStatus: "Complete",
				MissingInputs:    []string{"A", "B", "C", "D"},
				SkippedSLIs:      []string{"X", "Y", "Z", "W"},
			},
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

// mockStaticFetcher 는 항상 고정된 메트릭 값을 반환함 (대기 없음).
type mockStaticFetcher struct {
	values map[string]float64
}

func (m *mockStaticFetcher) Fetch(_ context.Context, at time.Time) (fetch.Sample, error) {
	return fetch.Sample{At: at, Values: m.values}, nil
}

// BenchmarkEngine_Execute 는 SLI 평가 파이프라인 전체(2회 fetch + evalSLI)를 벤치마킹함.
func BenchmarkEngine_Execute(b *testing.B) {
	values := map[string]float64{
		`controller_runtime_reconcile_total{controller="hello",result="success"}`: 100,
		`controller_runtime_reconcile_total{controller="hello",result="error"}`:   5,
	}
	fetcher := &mockStaticFetcher{values: values}
	writer := &mockWriter{}
	eng := New(fetcher, writer, nil)

	specs := []spec.SLISpec{
		{
			ID:    "reconcile_success",
			Title: "Reconcile Success Count",
			Inputs: []spec.MetricRef{
				{Key: `controller_runtime_reconcile_total{controller="hello",result="success"}`},
			},
			Compute: spec.ComputeSpec{Mode: spec.ComputeDelta},
			Judge: &spec.JudgeSpec{
				Rules: []spec.Rule{{Op: spec.OpGE, Target: 1, Level: spec.LevelFail}},
			},
		},
	}

	req := ExecuteRequest{
		Config: RunConfig{
			RunID:      "bench-run",
			StartedAt:  time.Unix(1000, 0),
			FinishedAt: time.Unix(1060, 0),
		},
		Specs:       specs,
		Reliability: &summary.Reliability{},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := eng.Execute(context.Background(), req)
		if err != nil {
			b.Fatal(err)
		}
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
