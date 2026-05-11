package gate

import (
	"math"
	"testing"
)

// TestDeltaPct_BaseZero_CurrentNonzero_ReturnsInf documents the deltaPct
// contract: base=0, cur≠0 returns +Inf. This is intentional — callers are
// responsible for guarding this case before JSON serialization.
// See evalRegressionCheck for the required guard pattern.
func TestDeltaPct_BaseZero_CurrentNonzero_ReturnsInf(t *testing.T) {
	if !math.IsInf(deltaPct(1, 0), 1) {
		t.Fatal("expected +Inf for deltaPct(1, 0) — guard contract changed")
	}
}

func TestDeltaPct_BothZero_ReturnsZero(t *testing.T) {
	if deltaPct(0, 0) != 0 {
		t.Fatal("expected 0 for deltaPct(0, 0)")
	}
}

func TestDeltaPct_Normal(t *testing.T) {
	got := deltaPct(6, 3)
	if got != 100.0 {
		t.Fatalf("expected 100.0, got %v", got)
	}
}
