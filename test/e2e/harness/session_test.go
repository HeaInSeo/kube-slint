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
	// 모의 fetcher 설정
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

// mockSnapshotFetcher 는 fetch.SnapshotFetcher 를 구현하는 테스트용 fetcher임.
type mockSnapshotFetcher struct {
	mockFetcher
	preFetched bool
}

func (m *mockSnapshotFetcher) PreFetch(_ context.Context) error {
	m.preFetched = true
	return nil
}

// TestSession_Start_PreFetch 는 Start()가 fetch.SnapshotFetcher 구현체에 대해
// PreFetch()를 호출하는지 검증함 (Gap G 해소 동작 확인).
func TestSession_Start_PreFetch(t *testing.T) {
	t.Setenv("SLINT_DISABLE_DISCOVERY", "1")
	sf := &mockSnapshotFetcher{}
	cfg := SessionConfig{
		Namespace: "ns",
		TestCase:  "test",
		Fetcher:   sf,
	}
	sess := NewSession(cfg)
	sess.Start()
	assert.True(t, sf.preFetched, "Start()는 fetcher가 SnapshotFetcher를 구현할 때 PreFetch()를 호출해야 함")
}

// TestSession_Start_NoPreFetch 는 일반 MetricsFetcher에서는 PreFetch가 호출되지 않음을 검증함.
func TestSession_Start_NoPreFetch(t *testing.T) {
	t.Setenv("SLINT_DISABLE_DISCOVERY", "1")
	mf := &mockFetcher{}
	cfg := SessionConfig{
		Namespace: "ns",
		TestCase:  "test",
		Fetcher:   mf,
	}
	sess := NewSession(cfg)
	// Start()가 패닉 없이 완료되고, mockFetcher는 SnapshotFetcher를 구현하지 않으므로 영향 없음
	sess.Start()
	assert.NotNil(t, sess.impl.started) // Start()가 정상적으로 시각을 기록했는지 확인
}

// ExampleSession_End는 권장되는 E2E 테스트 하네스 사용 패턴 예시임.
func ExampleSession_End() {
	var ctx context.Context // 유효한 context로 가정

	cfg := SessionConfig{
		Namespace: "operator-system",
		TestCase:  "E2E Integration Test",
		// Mode는 임시 리소스를 삭제할 시기를 결정함
		CleanupMode: "on-success",
	}

	sess := NewSession(cfg)

	// mockTestFailed는 테스트 프레임워크의 실패 상태를 나타냄
	var mockTestFailed bool

	// 1. Cleanup을 먼저 defer로 등록하여 마지막에 실행되도록 하고, MarkFailed로 테스트 상태를 동기화함
	defer func() {
		if mockTestFailed {
			sess.MarkFailed()
		}
		sess.Cleanup(ctx)
	}()

	// 2. 측정 창(window) 시작
	sess.Start()

	// 3. 오퍼레이터 활동 유발 및 수렴 대기...
}

func TestShouldRunCleanup(t *testing.T) {
	tests := []struct {
		name      string
		mode      string
		enabled   bool
		hasFailed bool
		want      bool
	}{
		{name: "ManualMode_NeverRuns", mode: "manual", enabled: true, hasFailed: false, want: false},
		{name: "ManualMode_NeverRuns_Fail", mode: "manual", enabled: true, hasFailed: true, want: false},

		{name: "EmptyMode_Enabled_IsAlways_Pass", mode: "", enabled: true, hasFailed: false, want: true},
		{name: "EmptyMode_Enabled_IsAlways_Fail", mode: "", enabled: true, hasFailed: true, want: true},
		{name: "EmptyMode_Disabled_NeverRuns", mode: "", enabled: false, hasFailed: false, want: false},

		// mode overrides enabled field logic locally
		{name: "AlwaysMode_Pass", mode: "always", enabled: false, hasFailed: false, want: true},
		{name: "AlwaysMode_Fail", mode: "always", enabled: true, hasFailed: true, want: true},

		{name: "OnSuccess_TestSucceeded", mode: "on-success", enabled: true, hasFailed: false, want: true},
		{name: "OnSuccess_TestFailed", mode: "on-success", enabled: true, hasFailed: true, want: false},

		{name: "OnFailure_TestSucceeded", mode: "on-failure", enabled: true, hasFailed: false, want: false},
		{name: "OnFailure_TestFailed", mode: "on-failure", enabled: true, hasFailed: true, want: true},

		{name: "UnknownMode_DefaultsToTrue", mode: "unknown-typo", enabled: true, hasFailed: false, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldRunCleanup(tt.mode, tt.enabled, tt.hasFailed)
			assert.Equal(t, tt.want, got)
		})
	}
}
