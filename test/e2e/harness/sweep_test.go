package harness

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSweepOrphansWithResult_Fallback(t *testing.T) {
	// 1. 잘못된 mode 입력 시 fallback 처리를 기록하는지 확인 (MissingGuard=false 즉 ns/runID 유효 시)
	cfg := SessionConfig{Namespace: "test-ns", RunID: "run-1"}
	sess := NewSession(cfg)

	// dummy kubectl 실행을 방지하기 위해 kubectl이 없거나 에러가 나더라도
	// fallbacks 기록 상태는 그 전에 처리되므로 assert 가능해야 함
	// kubectl이 실패하면 err가 반환되지만 결과 구조체(SweepResult)는 채워짐
	res, err := sess.SweepOrphansWithResult(context.Background(), OrphanSweepOptions{
		Enabled: true,
		Mode:    "invalid-mode",
	})

	// kubectl 에러가 발생하더라도 (또는 발견 못하더라도) 구조체 검사
	_ = err

	assert.Equal(t, "invalid_mode", res.Apply.FallbackReason)
	assert.True(t, res.Apply.ModeFallback)
	assert.Equal(t, "report-only", res.Apply.ModeEffective)

	assert.Contains(t, res.Warnings[0], "invalid mode \"invalid-mode\" provided, falling back to report-only")
}

func TestSweepOrphansWithResult_MissingGuard(t *testing.T) {
	// 2. runID가 없어 Guard에 걸릴 때 결과에 스킵 이유가 나오는지 확인
	cfg := SessionConfig{RunID: ""}
	sess := NewSession(cfg)
	res, err := sess.SweepOrphansWithResult(context.Background(), OrphanSweepOptions{
		Enabled: true,
		Mode:    "report-only",
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, res.Summary.SkippedByReason["missing_guard"])
	assert.Contains(t, res.Warnings[0], "missing namespace or run-id")
}

func TestSweepOrphansWithResult_JSON(t *testing.T) {
	res := SweepResult{
		SchemaVersion: "v1.0",
		StartedAt:     time.Now(),
		Request: SweepRequest{
			Namespace: "test-ns",
			Limit:     10,
		},
	}

	var buf bytes.Buffer
	err := WriteSweepResultJSON(&buf, res)
	assert.NoError(t, err)

	jsonOutput := buf.String()
	assert.Contains(t, jsonOutput, `"schemaVersion": "v1.0"`)
	assert.Contains(t, jsonOutput, `"namespace": "test-ns"`)
	assert.Contains(t, jsonOutput, `"limit": 10`)
}
