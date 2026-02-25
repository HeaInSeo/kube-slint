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
	// ... doWork() ...
	// if err != nil { mockTestFailed = true; return }

	// 4. 측정 창을 종료하고 평가를 시작함.
	// End() 실행 중 CheckStrictness/Gating 규칙이 적용됨.
	_, err := sess.End(ctx)
	if err != nil {
		mockTestFailed = true
		// e.g. Expect(err).ToNot(HaveOccurred()) in Ginkgo
	}

	// 5. 이전 실행에서 남겨진 고아 리소스 정리 (선택 사항)
	// 예기치 않은 삭제를 막기 위해 "report-only" 모드 사용을 권장함.
	_ = sess.SweepOrphans(ctx, OrphanSweepOptions{Enabled: true, Mode: "report-only"})
}

func TestSession_SweepOrphans_Disabled(t *testing.T) {
	cfg := SessionConfig{Namespace: "ns", RunID: "run-1"}
	sess := NewSession(cfg)
	// 비활성화 시 패닉이나 에러가 발생하면 안 됨
	err := sess.SweepOrphans(context.Background(), OrphanSweepOptions{Enabled: false})
	assert.NoError(t, err)
}

func TestSession_SweepOrphans_MissingGuard(t *testing.T) {
	// 다중 실행 안전을 위해 Namespace가 없으면 에러가 아닌 생략(skip) 처리되어야 함
	cfg := SessionConfig{RunID: "run-1"}
	sess := NewSession(cfg)
	err := sess.SweepOrphans(context.Background(), OrphanSweepOptions{Enabled: true, Mode: "report-only"})
	assert.NoError(t, err)
}
