package kubeutil

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsPrometheusOperatorCRDsInstalled_LogsUnderlyingErrorInsteadOfSwallowingIt
// mirrors TestIsCertManagerCRDsInstalled_LogsUnderlyingErrorInsteadOfSwallowingIt
// — same finding, same fix, sibling function.
func TestIsPrometheusOperatorCRDsInstalled_LogsUnderlyingErrorInsteadOfSwallowingIt(t *testing.T) {
	log := &fakeLogger{}
	runner := fakeFailingRunner{err: errors.New("connection refused")}

	got := IsPrometheusOperatorCRDsInstalled(context.Background(), log, runner)

	assert.False(t, got)
	found := false
	for _, m := range log.messages {
		if strings.Contains(m, "kubectl get crds failed") {
			found = true
		}
	}
	assert.True(t, found, "expected the underlying kubectl error to be logged, got: %v", log.messages)
}
