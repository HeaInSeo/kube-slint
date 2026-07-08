package kubeutil

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/slo"
	"github.com/stretchr/testify/assert"
)

type fakeFailingRunner struct {
	err error
}

func (f fakeFailingRunner) Run(ctx context.Context, logger slo.Logger, cmd *exec.Cmd) (string, error) {
	return "", f.err
}

type fakeLogger struct {
	messages []string
}

func (f *fakeLogger) Logf(format string, args ...any) {
	f.messages = append(f.messages, format)
}

// TestIsCertManagerCRDsInstalled_LogsUnderlyingErrorInsteadOfSwallowingIt is a
// regression test for a finding from pre-release-adversarial-review
// (2026-07-08): a failed `kubectl get crds` call (no cluster access,
// permission denied, etc.) was silently converted to `false`, making it
// indistinguishable from "CRDs genuinely not installed."
func TestIsCertManagerCRDsInstalled_LogsUnderlyingErrorInsteadOfSwallowingIt(t *testing.T) {
	log := &fakeLogger{}
	runner := fakeFailingRunner{err: errors.New("connection refused")}

	got := IsCertManagerCRDsInstalled(context.Background(), log, runner)

	assert.False(t, got)
	found := false
	for _, m := range log.messages {
		if strings.Contains(m, "kubectl get crds failed") {
			found = true
		}
	}
	assert.True(t, found, "expected the underlying kubectl error to be logged, got: %v", log.messages)
}
