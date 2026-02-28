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
	"github.com/HeaInSeo/kube-slint/test/e2e/harness"
)

type mockMetricsFetcher struct {
	serverURL string
}

func (m *mockMetricsFetcher) Fetch(ctx context.Context, at time.Time) (fetch.Sample, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.serverURL, nil)
	if err != nil {
		return fetch.Sample{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fetch.Sample{}, err
	}
	defer resp.Body.Close()
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
			fmt.Sscanf(parts[1], "%f", &v)
			values[parts[0]] = v
		}
	}
	return fetch.Sample{At: at, Values: values}, nil
}

func TestHarnessIntegration(t *testing.T) {
	mockResponseText := `
operator_up 1.0
workqueue_adds_total{name="my-controller"} 42.0
`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(mockResponseText))
	}))
	defer ts.Close()

	cfg := harness.SessionConfig{
		Namespace:          "test-namespace",
		MetricsServiceName: "mock-metrics",
		TestCase:           "Consumer-Library-Integration",
		Suite:              "E2E-Replacer",
		RunID:              "mock-run-123",
		Specs: []spec.SLISpec{
			{
				ID:      "up",
				Inputs:  []spec.MetricRef{spec.UnsafePromKey("operator_up")},
				Compute: spec.ComputeSpec{Mode: spec.ComputeSingle},
				Judge: &spec.JudgeSpec{
					Rules: []spec.Rule{
						{Metric: "value", Op: spec.OpLT, Target: 1.0, Level: spec.LevelFail},
					},
				},
			},
			{
				ID:      "wq",
				Inputs:  []spec.MetricRef{spec.UnsafePromKey(`workqueue_adds_total{name="my-controller"}`)},
				Compute: spec.ComputeSpec{Mode: spec.ComputeSingle},
				Judge: &spec.JudgeSpec{
					Rules: []spec.Rule{
						{Metric: "value", Op: spec.OpLT, Target: 42.0, Level: spec.LevelFail},
					},
				},
			},
		},
		Fetcher: &mockMetricsFetcher{serverURL: ts.URL},
	}
	session := harness.NewSession(cfg)
	session.Start()
	time.Sleep(10 * time.Millisecond)
	summary, err := session.End(context.Background())
	if err != nil {
		t.Fatalf("Session.End failed: %v", err)
	}

	if len(summary.Results) != 2 {
		t.Fatalf("Expected 2 evaluated results, got %d", len(summary.Results))
	}

	for _, res := range summary.Results {
		if res.Status != "pass" {
			t.Errorf("Expected metric %s to pass, got %s: %s", res.ID, res.Status, res.Reason)
		}
		if res.ID == "up" && (res.Value == nil || *res.Value != 1.0) {
			t.Errorf("Expected up=1.0")
		}
		if res.ID == "wq" && (res.Value == nil || *res.Value != 42.0) {
			t.Errorf("Expected wq=42.0")
		}
	}
}
