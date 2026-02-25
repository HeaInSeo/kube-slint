package harness

import (
	"context"
	"testing"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/slo/fetch"
	"github.com/stretchr/testify/assert"
)

func TestNewSession(t *testing.T) {
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := SessionConfig{
		Namespace:          "test-ns",
		MetricsServiceName: "test-svc",
		TestCase:           "TestCase A",
		Suite:              "e2e",
		RunID:              "run-123",
		Tags:               map[string]string{"env": "ci"},
		Now:                func() time.Time { return now },
	}

	sess := NewSession(cfg)

	assert.Equal(t, "test-ns", sess.impl.Config.Namespace)
	assert.Equal(t, "run-123", sess.impl.RunID)
	assert.Equal(t, "e2e", sess.impl.Tags["suite"])
	assert.Equal(t, "TestCase A", sess.impl.Tags["test_case"])
	assert.Equal(t, "ci", sess.impl.Tags["env"])
	assert.NotNil(t, sess.impl.writer)
}

func TestSession_AutoRunID(t *testing.T) {
	now := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	cfg := SessionConfig{
		Namespace: "ns",
		Now:       func() time.Time { return now },
	}

	sess := NewSession(cfg)
	assert.Equal(t, "local-1704103200", sess.impl.RunID)
}

func TestSession_End(t *testing.T) {
	// Mock fetcher
	mockFetcher := &mockFetcher{}

	cfg := SessionConfig{
		Namespace: "ns",
		TestCase:  "test",
		Fetcher:   mockFetcher,
	}
	sess := NewSession(cfg)
	sess.Start()

	summary, err := sess.End(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, summary)
	assert.Equal(t, "v4.4", summary.Config.Format)
}

type mockFetcher struct{}

func (m *mockFetcher) Fetch(ctx context.Context, at time.Time) (fetch.Sample, error) {

	return fetch.Sample{
		At:     at,
		Values: map[string]float64{"foo": 1.0},
	}, nil
}

// ExampleSession_End shows the recommended way to use the harness in E2E tests, particularly for Cleanup.
// 권장되는 E2E 테스트 하네스 사용 패턴 예시입니다.
func ExampleSession_End() {
	var ctx context.Context // assume valid context

	cfg := SessionConfig{
		Namespace: "operator-system",
		TestCase:  "E2E Integration Test",
		// Mode determines when to delete temporary resources
		CleanupMode: "on-success",
	}

	sess := NewSession(cfg)

	// mockTestFailed represents the test framework's failure state (e.g., Ginkgo's CurrentSpecReport().Failed())
	// mockTestFailed는 테스트 프레임워크의 실패 상태를 나타냅니다.
	var mockTestFailed bool

	// 1. defer Cleanup first so it runs at the end. Use MarkFailed to sync test state.
	defer func() {
		if mockTestFailed {
			sess.MarkFailed()
		}
		sess.Cleanup(ctx)
	}()

	// 2. Start the measurement window
	sess.Start()

	// 3. Do operator interactions, wait for convergence...
	// ... doWork() ...
	// if err != nil { mockTestFailed = true; return }

	// 4. End the measurement window and trigger evaluations.
	// CheckStrictness/Gating is enforced during End().
	_, err := sess.End(ctx)
	if err != nil {
		mockTestFailed = true
		// e.g. Expect(err).ToNot(HaveOccurred()) in Ginkgo
	}
}
