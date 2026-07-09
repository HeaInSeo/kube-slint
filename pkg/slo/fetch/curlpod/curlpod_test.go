package curlpod

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/slo"
)

// scriptedRunner dispatches a canned response based on the kubectl
// subcommand, letting a single fake drive a full RunOnce -> WaitDone ->
// Logs -> cleanup sequence.
type scriptedRunner struct {
	phase    string
	phaseErr error
	logs     string
	logsErr  error

	commands []string
}

func (r *scriptedRunner) Run(_ context.Context, _ slo.Logger, cmd *exec.Cmd) (string, error) {
	r.commands = append(r.commands, strings.Join(cmd.Args, " "))
	for _, a := range cmd.Args {
		if a == "jsonpath={.status.phase}" {
			return r.phase, r.phaseErr
		}
	}
	if len(cmd.Args) > 1 && cmd.Args[1] == "logs" {
		return r.logs, r.logsErr
	}
	return "", nil
}

func newTestCurlPod(r *scriptedRunner) *CurlPod {
	client := New(nil, r)
	return &CurlPod{
		Client:             client,
		Namespace:          "ns",
		MetricsServiceName: "metrics-svc",
		ServiceAccountName: "scraper-sa",
	}
}

func TestCurlPodRun_Succeeded_ReturnsLogs(t *testing.T) {
	r := &scriptedRunner{phase: "Succeeded", logs: "reconcile_total 5\n"}
	pod := newTestCurlPod(r)

	out, err := pod.Run(context.Background(), time.Second, time.Second)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if out != "reconcile_total 5\n" {
		t.Fatalf("expected logs to be returned verbatim, got: %q", out)
	}
}

// TestCurlPodRun_Failed_ReturnsErrorNotLogs reproduces the finding from the
// third pre-release-adversarial-review pass (2026-07-09): a curl pod that
// reaches phase Failed (e.g. curl --fail-with-body exiting non-zero on an
// RBAC 403) used to have its raw log body returned as if it were a
// successful measurement (out, nil), which the caller would then feed
// straight into the Prometheus text parser. Run must now return an error
// instead of the raw output on a Failed phase.
func TestCurlPodRun_Failed_ReturnsErrorNotLogs(t *testing.T) {
	r := &scriptedRunner{
		phase: "Failed",
		logs:  `{"kind":"Status","status":"Failure","message":"Forbidden","code":403}`,
	}
	pod := newTestCurlPod(r)

	out, err := pod.Run(context.Background(), time.Second, time.Second)
	if err == nil {
		t.Fatal("expected an error for a Failed pod phase, got nil")
	}
	if out != "" {
		t.Fatalf("expected empty output on failure (must not be treated as a measurement), got: %q", out)
	}
	if !errors.Is(err, ErrPodFailed) {
		t.Fatalf("expected errors.Is(err, ErrPodFailed) to be true, got: %v", err)
	}
}

func TestCurlPodRun_Failed_ErrorIncludesRedactedPodOutput(t *testing.T) {
	r := &scriptedRunner{
		phase: "Failed",
		logs:  `error: Authorization: Bearer super-secret-token-value`,
	}
	pod := newTestCurlPod(r)

	_, err := pod.Run(context.Background(), time.Second, time.Second)
	if err == nil {
		t.Fatal("expected an error for a Failed pod phase, got nil")
	}
	if strings.Contains(err.Error(), "super-secret-token-value") {
		t.Fatalf("pod output embedded in the error must be redacted, got: %v", err)
	}
	if !strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatalf("expected the redacted marker in the error, got: %v", err)
	}
}

func TestCurlPodRun_Failed_StillCleansUpPod(t *testing.T) {
	r := &scriptedRunner{phase: "Failed", logs: "some output"}
	pod := newTestCurlPod(r)

	if _, err := pod.Run(context.Background(), time.Second, time.Second); err == nil {
		t.Fatal("expected an error")
	}

	found := false
	for _, c := range r.commands {
		if strings.HasPrefix(c, "kubectl delete pod ") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a cleanup delete even on a Failed pod, commands: %v", r.commands)
	}
}

func TestCurlPodRun_KubectlWaitError_DoesNotFetchLogs(t *testing.T) {
	r := &scriptedRunner{phaseErr: context.DeadlineExceeded, logs: "should never be returned"}
	pod := newTestCurlPod(r)

	out, err := pod.Run(context.Background(), time.Second, time.Second)
	if err == nil {
		t.Fatal("expected an error")
	}
	if out != "" {
		t.Fatalf("expected empty output, got: %q", out)
	}
	if errors.Is(err, ErrPodFailed) {
		t.Fatal("a kubectl/context error must not be reported as ErrPodFailed")
	}
	for _, c := range r.commands {
		if len(c) >= 4 && c[:4] == "logs" {
			t.Fatal("must not fetch logs when the pod never reached a terminal phase")
		}
	}
}
