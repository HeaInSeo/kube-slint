package service_test

import (
	"path/filepath"
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/dataplane/service"
	"github.com/HeaInSeo/kube-slint/pkg/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAnalyze_KindHelloOperatorFixture runs the analyzer against the real
// examples/kind-hello-operator/manifests/ fixture. Known properties of that
// fixture (confirmed by reading the manifest directly): a metrics
// containerPort + matching Service exist; readinessProbe targets /healthz
// (not /readyz); there is no terminationGracePeriodSeconds; there is no
// ServiceMonitor anywhere in the example. (KSL-DP-003/005 no longer run —
// removed as duplicates of kube-linter's probe/resource checks — so the
// fixture's missing livenessProbe and fully-set resources no longer produce
// findings here.)
func TestAnalyze_KindHelloOperatorFixture(t *testing.T) {
	fixtureDir := filepath.Join("..", "..", "..", "examples", "kind-hello-operator", "manifests")

	rep, warnings, err := service.Analyze(fixtureDir, "test")
	require.NoError(t, err)
	assert.Empty(t, warnings)

	assert.Equal(t, 0, rep.Summary.ErrorCount, "no remaining check produces an error for this fixture")
	assert.Equal(t, 3, rep.Summary.WarningCount, "expected exactly 3 warnings (probe path, ServiceMonitor, grace period)")

	byRule := map[string]int{}
	for _, f := range rep.Findings {
		byRule[f.RuleID]++
	}

	assert.Zero(t, byRule["KSL-DP-001"], "metrics port exists in the fixture")
	assert.Equal(t, 1, byRule["KSL-DP-002"], "readinessProbe path /healthz != /readyz")
	assert.Equal(t, 1, byRule["KSL-DP-004"], "no ServiceMonitor matches the metrics Service (Service itself matches, so no error half)")
	assert.Equal(t, 1, byRule["KSL-DP-006"], "terminationGracePeriodSeconds is not set")

	for _, f := range rep.Findings {
		if f.RuleID == "KSL-DP-004" {
			assert.Equal(t, report.SeverityWarning, f.Severity, "Service already matches, so KSL-DP-004 here must be the ServiceMonitor warning, not the Service error")
		}
	}
}
