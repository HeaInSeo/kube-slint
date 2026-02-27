package harness

import (
	"testing"

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
}
