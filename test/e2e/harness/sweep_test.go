package harness

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "no command\n")
		os.Exit(2)
	}

	command, args := args[0], args[1:]
	if command == "kubectl" {
		switch args[0] {
		case "get":
			fmt.Print("pod1,run-1,2023-01-01T00:00:00Z\npod2,run-1,2023-01-01T00:00:00Z\n")
			os.Exit(0)
		case modeDelete:
			if args[2] == "failpod" {
				fmt.Fprintf(os.Stderr, "error from server\n")
				os.Exit(1)
			}
			fmt.Printf("pod \"%s\" deleted\n", args[2])
			os.Exit(0)
		}
	}
	os.Exit(1)
}

func fakeExecCommand(ctx context.Context, command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func TestSweepDeletes_SuccessCount(t *testing.T) {
	execCommandContext = fakeExecCommand
	defer func() { execCommandContext = exec.CommandContext }()

	opts := OrphanSweepOptions{
		Enabled: true,
		Mode:    modeDelete,
		Limit:   10,
	}
	res := initSweepResult(opts, time.Now())
	res.Apply.ModeEffective = modeDelete

	// mock items
	res.Items = append(res.Items, SweepItem{Name: "pod1", Action: "would-delete"})
	res.Items = append(res.Items, SweepItem{Name: "pod2", Action: "would-delete"})
	targetNames := []string{"pod1", "pod2"}

	err := applySweepDeletes(context.Background(), "default", targetNames, &res)
	assert.NoError(t, err)

	assert.Equal(t, 2, res.Summary.Deleted, "Deleted count correctly matches 2 successful deletions")
	assert.Equal(t, "deleted", res.Items[0].Action)
	assert.Equal(t, "deleted", res.Items[1].Action)
}

func TestSweepDeletes_FailureCount(t *testing.T) {
	execCommandContext = fakeExecCommand
	defer func() { execCommandContext = exec.CommandContext }()

	opts := OrphanSweepOptions{
		Enabled: true,
		Mode:    modeDelete,
		Limit:   10,
	}
	res := initSweepResult(opts, time.Now())
	res.Apply.ModeEffective = modeDelete

	res.Items = append(res.Items, SweepItem{Name: "pod1", Action: "would-delete"})
	res.Items = append(res.Items, SweepItem{Name: "failpod", Action: "would-delete"})
	targetNames := []string{"pod1", "failpod"}

	err := applySweepDeletes(context.Background(), "default", targetNames, &res)
	assert.Error(t, err)

	assert.Equal(t, 1, res.Summary.Deleted, "Deleted count matches 1 success")
	assert.Equal(t, 1, res.Summary.DeleteError, "DeleteError count matches 1 failure")

	assert.Equal(t, "deleted", res.Items[0].Action)
	assert.Equal(t, "delete-error", res.Items[1].Action)
}

func TestSweepOrphansWithResult_Fallback(t *testing.T) {
	cfg := SessionConfig{Namespace: "test-ns", RunID: "run-1"}
	sess := NewSession(cfg)

	res, err := sess.SweepOrphansWithResult(context.Background(), OrphanSweepOptions{
		Enabled: true,
		Mode:    "invalid-mode",
	})
	_ = err

	assert.Equal(t, "invalid_mode", res.Apply.FallbackReason)
	assert.True(t, res.Apply.ModeFallback)
	assert.Equal(t, "report-only", res.Apply.ModeEffective)
	assert.Contains(t, res.Warnings[0], "invalid mode \"invalid-mode\" provided, falling back to report-only")
}

func TestSweepOrphansWithResult_EmptyMode(t *testing.T) {
	// 빈 mode("") 정상 기본값 처리 테스트
	cfg := SessionConfig{Namespace: "test-ns", RunID: "run-1"}
	sess := NewSession(cfg)

	res, err := sess.SweepOrphansWithResult(context.Background(), OrphanSweepOptions{
		Enabled: true,
		Mode:    "",
	})
	_ = err

	assert.False(t, res.Apply.ModeFallback, "empty mode is not an invalid mode fallback")
	assert.Equal(t, "", res.Apply.FallbackReason)
	assert.Equal(t, "report-only", res.Apply.ModeEffective)
	assert.Empty(t, res.Warnings, "empty mode should not generate warnings")
}

func TestSweepCandidate_TimestampParseFailedAndMaxAge(t *testing.T) {
	opts := OrphanSweepOptions{
		Enabled: true,
		Mode:    "report-only",
		Limit:   10,
		MaxAge:  1 * time.Hour,
	}
	res := initSweepResult(opts, time.Now())
	targetNames := []string{}
	hitLimit := 0
	evaluateSweepCandidate(
		"pod-bad-time,run-old,invalid_timestamp",
		opts, "default", time.Now(), &res, &targetNames, &hitLimit,
	)

	assert.Equal(t, 1, res.Summary.Skipped)
	assert.Equal(t, 1, res.Summary.SkippedByReason["timestamp_parse_failed"])
	assert.Equal(t, 1, len(res.Items))
	assert.Equal(t, "skipped", res.Items[0].Action)
	assert.Equal(t, "timestamp_parse_failed", res.Items[0].Reason)
	// warning must be appended
	assert.Len(t, res.Warnings, 1)
	assert.Contains(t, res.Warnings[0], "failed to parse creation timestamp")
}

func TestSweepCandidate_LimitExceeded(t *testing.T) {
	opts := OrphanSweepOptions{
		Enabled: true,
		Mode:    "report-only",
		Limit:   1, // strict limit
		MaxAge:  0,
	}
	res := initSweepResult(opts, time.Now())

	// mock that we already reached the limit
	targetNames := []string{"pod1"}
	hitLimit := 0

	// evaluation for pod2 should hit limit
	evaluateSweepCandidate("pod2,run-old,2021-01-01T00:00:00Z", opts, "default", time.Now(), &res, &targetNames, &hitLimit)

	assert.Equal(t, 1, res.Summary.Skipped)
	assert.Equal(t, 1, res.Summary.SkippedByReason["limit_exceeded"])
	assert.Equal(t, 1, hitLimit)
	assert.Equal(t, "skipped", res.Items[0].Action)
	assert.Equal(t, "limit_exceeded", res.Items[0].Reason)
}

func TestSweepOrphansWithResult_MissingGuard(t *testing.T) {
	cfg := SessionConfig{RunID: ""} // guard condition fail
	sess := NewSession(cfg)
	res, err := sess.SweepOrphansWithResult(context.Background(), OrphanSweepOptions{
		Enabled: true,
		Mode:    "report-only",
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, res.Summary.Skipped) // missing_guard creates 1 skipped
	assert.Equal(t, 1, res.Summary.SkippedByReason["missing_guard"])
	assert.Contains(t, res.Warnings[0], "missing namespace or run-id")
}
