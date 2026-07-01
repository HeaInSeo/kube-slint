package slint

import (
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/slo/spec"
	"github.com/stretchr/testify/assert"
)

func TestDefaultSpecs_Smoke(t *testing.T) {
	specs := DefaultV3Specs()

	assert.NotEmpty(t, specs, "DefaultV3Specs should return at least one spec")

	for i, s := range specs {
		assert.NotEmpty(t, s.ID, "Spec at index %d must have an ID", i)
		assert.NotEmpty(t, s.Inputs, "Spec %s must have inputs defined", s.ID)
		if s.Judge != nil {
			assert.NotEmpty(t, s.Judge.Rules, "Spec %s with Judge must have Rules defined", s.ID)
		}
	}

	// Check specific expected baseline
	hasReconcileTotal := false
	for _, s := range specs {
		if s.ID == "reconcile_total_delta" {
			hasReconcileTotal = true
			break
		}
	}
	assert.True(t, hasReconcileTotal, "reconcile_total_delta should be provided in baseline defaults")

	// workqueue_depth_end must use ComputeEnd so it reflects the end-of-window snapshot,
	// not the start snapshot (ComputeSingle/ComputeStart).
	for _, s := range specs {
		if s.ID == "workqueue_depth_end" {
			assert.Equal(t, spec.ComputeEnd, s.Compute.Mode,
				"workqueue_depth_end must use ComputeEnd to measure queue depth at window close")
		}
	}
}
