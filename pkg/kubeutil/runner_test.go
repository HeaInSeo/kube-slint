package kubeutil

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

func TestDefaultRunner_RedactsSecretsInCommandLogAndError(t *testing.T) {
	logger := &recordingLogger{}
	cmd := exec.Command("sh", "-c", "echo token=supersecret >&2; echo 'Authorization: Bearer abc123' >&2; exit 1")

	_, err := (DefaultRunner{}).Run(context.Background(), logger, cmd)
	if err == nil {
		t.Fatal("expected command error")
	}

	logs := strings.Join(logger.lines, "\n")
	if strings.Contains(logs, "supersecret") || strings.Contains(logs, "abc123") {
		t.Fatalf("secret leaked in logs: %s", logs)
	}
	errText := err.Error()
	if strings.Contains(errText, "supersecret") || strings.Contains(errText, "abc123") {
		t.Fatalf("secret leaked in error: %s", errText)
	}
	if !strings.Contains(logs, "[REDACTED]") || !strings.Contains(errText, "[REDACTED]") {
		t.Fatalf("expected redaction markers in logs and error, logs=%q err=%q", logs, errText)
	}
}

type recordingLogger struct {
	lines []string
}

func (l *recordingLogger) Logf(format string, args ...any) {
	l.lines = append(l.lines, strings.TrimSpace(formatMessage(format, args...)))
}

func formatMessage(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}
