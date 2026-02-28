package e2e_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/slo/fetch"
	"github.com/HeaInSeo/kube-slint/pkg/slo/spec"
	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
	"github.com/HeaInSeo/kube-slint/test/e2e/harness"
)

type mockMetricsFetcher struct {
	serverURL string
	callCount int
}

func (m *mockMetricsFetcher) Fetch(ctx context.Context, at time.Time) (fetch.Sample, error) {
	m.callCount++

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.serverURL, nil)
	if err != nil {
		return fetch.Sample{}, err
	}

	// Add custom header to inform the mock server about the call sequence
	req.Header.Set("X-Call-Count", fmt.Sprintf("%d", m.callCount))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// Network error simulation
		return fetch.Sample{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fetch.Sample{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fetch.Sample{}, err
	}

	values := make(map[string]float64)
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			var v float64
			_, _ = fmt.Sscanf(parts[1], "%f", &v)
			values[parts[0]] = v
		}
	}
	return fetch.Sample{At: at, Values: values}, nil
}

//nolint:gocognit,lll,gocyclo // Test suites with inline setups inherently trigger these limits
func TestHarnessIntegration_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		handler  http.HandlerFunc
		specs    []spec.SLISpec
		validate func(t *testing.T, s *summary.Summary, sessionErr error)
	}{
		{
			name: "Happy Path - Single Metric Computations",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				_, _ = w.Write([]byte(`operator_up 1.0` + "\n" + `workqueue_adds_total{name="my-controller"} 42.0`))
			},
			specs: []spec.SLISpec{
				{
					ID:      "up",
					Inputs:  []spec.MetricRef{spec.UnsafePromKey("operator_up")},
					Compute: spec.ComputeSpec{Mode: spec.ComputeSingle},
					Judge:   &spec.JudgeSpec{Rules: []spec.Rule{{Metric: "value", Op: spec.OpLT, Target: 1.0, Level: spec.LevelFail}}},
				},
				{
					ID:      "wq",
					Inputs:  []spec.MetricRef{spec.UnsafePromKey(`workqueue_adds_total{name="my-controller"}`)},
					Compute: spec.ComputeSpec{Mode: spec.ComputeSingle},
					Judge:   &spec.JudgeSpec{Rules: []spec.Rule{{Metric: "value", Op: spec.OpLT, Target: 42.0, Level: spec.LevelFail}}},
				},
			},
			validate: func(t *testing.T, s *summary.Summary, err error) {
				if err != nil {
					t.Fatalf("Expected nil error, got %v", err)
				}
				if len(s.Results) != 2 {
					t.Fatalf("Expected 2 results")
				}

				for _, res := range s.Results {
					if res.ID == "up" {
						if res.Status != summary.StatusPass {
							t.Errorf("Expected PASS, got %s", res.Status)
						}
						if res.Value == nil || *res.Value != 1.0 {
							t.Errorf("Expected 1.0, got %v", res.Value)
						}
					}
					if res.ID == "wq" {
						if res.Status != summary.StatusPass {
							t.Errorf("Expected PASS, got %s: %s", res.Status, res.Reason)
						}
						if res.Value == nil || *res.Value != 42.0 {
							t.Errorf("Expected 42.0, got %v", res.Value)
						}
					}
				}
			},
		},
		{
			name: "Missing Metric Input",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				_, _ = w.Write([]byte(`other_metric 123.0`)) // operator_up is entirely missing
			},
			specs: []spec.SLISpec{
				{
					ID:      "up",
					Inputs:  []spec.MetricRef{spec.UnsafePromKey("operator_up")},
					Compute: spec.ComputeSpec{Mode: spec.ComputeSingle},
				},
			},
			validate: func(t *testing.T, s *summary.Summary, err error) {
				if err != nil {
					t.Fatalf("Expected nil error, got %v", err)
				}
				if len(s.Results) != 1 {
					t.Fatalf("Expected 1 result")
				}

				res := s.Results[0]
				if res.Status != summary.StatusSkip {
					t.Errorf("Expected status SKIP for missing metric, got %s", res.Status)
				}
				if !strings.Contains(res.Reason, "missing input metrics") {
					t.Errorf("Expected missing metric reason, got %q", res.Reason)
				}
			},
		},
		{
			name: "Network Fetch Error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError) // Simulate HTTP 500 error
			},
			specs: []spec.SLISpec{
				{
					ID:      "up",
					Inputs:  []spec.MetricRef{spec.UnsafePromKey("operator_up")},
					Compute: spec.ComputeSpec{Mode: spec.ComputeSingle},
				},
			},
			validate: func(t *testing.T, s *summary.Summary, err error) {
				// Evaluation usually succeeds partially but generates a summary detailing blocked status
				// due to missing scrape snapshot because of the 500 Network error.
				if err != nil {
					t.Fatalf("Expected session engine to swallow errors and emit diagnostic summary, got %v", err)
				}

				// Expected to evaluate as block or skip since the input values were fully missing due to scrape failure.
				// In some cases, Results might be empty if collection failed before initialization.
				if len(s.Results) > 0 {
					res := s.Results[0]
					if res.Status != summary.StatusSkip && res.Status != summary.StatusBlock {
						t.Errorf("Expected Skip or Block status from fetch err, got %s", res.Status)
					}
				}

				if s.Reliability.EvaluationStatus != "" && s.Reliability.EvaluationStatus != "Partial" && s.Reliability.EvaluationStatus != "Failed" {
					t.Errorf("Expected reliability to highlight partial/failed eval or empty, got %q", s.Reliability.EvaluationStatus)
				}
				if s.Reliability.CollectionStatus != "Partial" && s.Reliability.CollectionStatus != "Failed" {
					t.Errorf("Expected reliability to highlight partial/failed collection, got %s", s.Reliability.CollectionStatus)
				}
			},
		},
		{
			name: "Delta Path Scenario",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				countStr := r.Header.Get("X-Call-Count")
				if countStr == "1" {
					_, _ = w.Write([]byte(`events_processed_total 10.0`)) // Start snapshot
				} else {
					_, _ = w.Write([]byte(`events_processed_total 25.0`)) // End snapshot (delta should be 15)
				}
			},
			specs: []spec.SLISpec{
				{
					ID:      "proc",
					Inputs:  []spec.MetricRef{spec.UnsafePromKey("events_processed_total")},
					Compute: spec.ComputeSpec{Mode: spec.ComputeDelta},
					// We expect Delta = 15. The rule below says if value < 10, fail it. Since 15 > 10, it should pass.
					Judge: &spec.JudgeSpec{Rules: []spec.Rule{{Metric: "value", Op: spec.OpLT, Target: 10.0, Level: spec.LevelFail}}},
				},
			},
			validate: func(t *testing.T, s *summary.Summary, err error) {
				if err != nil {
					t.Fatalf("Expected nil error, got %v", err)
				}
				if len(s.Results) != 1 {
					t.Fatalf("Expected 1 result")
				}

				res := s.Results[0]
				if res.Status != summary.StatusPass {
					t.Errorf("Expected PASS status for delta=15, got %s: %s", res.Status, res.Reason)
				}
				if res.Value == nil || *res.Value != 15.0 {
					t.Errorf("Expected computed delta to be 15.0, got %v", res.Value)
				}
			},
		},
		{
			name: "Multi-metric Mixed Result",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				// operator_up=1.0 (pass), error_rate=5.0 (fail rule)
				_, _ = w.Write([]byte(`operator_up 1.0` + "\n" + `error_rate 5.0`))
			},
			specs: []spec.SLISpec{
				{
					ID:      "up",
					Inputs:  []spec.MetricRef{spec.UnsafePromKey("operator_up")},
					Compute: spec.ComputeSpec{Mode: spec.ComputeSingle},
					Judge:   &spec.JudgeSpec{Rules: []spec.Rule{{Metric: "value", Op: spec.OpLT, Target: 1.0, Level: spec.LevelFail}}}, // PASSes since 1.0 not < 1.0
				},
				{
					ID:      "err",
					Inputs:  []spec.MetricRef{spec.UnsafePromKey("error_rate")},
					Compute: spec.ComputeSpec{Mode: spec.ComputeSingle},
					Judge:   &spec.JudgeSpec{Rules: []spec.Rule{{Metric: "value", Op: spec.OpGT, Target: 2.0, Level: spec.LevelFail}}}, // FAILs since 5.0 > 2.0
				},
			},
			validate: func(t *testing.T, s *summary.Summary, err error) {
				if err != nil {
					t.Fatalf("Expected nil error, got %v", err)
				}
				if len(s.Results) != 2 {
					t.Fatalf("Expected 2 results")
				}

				for _, res := range s.Results {
					if res.ID == "up" {
						if res.Status != summary.StatusPass {
							t.Errorf("Expected up metric to PASS, got %s", res.Status)
						}
					}
					if res.ID == "err" {
						if res.Status != summary.StatusFail {
							t.Errorf("Expected err metric to FAIL, got %s", res.Status)
						}
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(tt.handler)
			defer ts.Close()

			cfg := harness.SessionConfig{
				Namespace:          "test-namespace",
				MetricsServiceName: "mock-metrics",
				TestCase:           tt.name,
				Suite:              "E2E-Replacer-Table",
				RunID:              "mock-run-456",
				Specs:              tt.specs,
				Fetcher:            &mockMetricsFetcher{serverURL: ts.URL},
				// Disable artifacts to avoid bloating test dir with jsons
				ArtifactsDir: "",
			}

			session := harness.NewSession(cfg)
			session.Start()
			time.Sleep(5 * time.Millisecond) // Ensure the Start time registers clearly distinct from End time logically
			summary, err := session.End(context.Background())

			tt.validate(t, summary, err)
		})
	}
}
