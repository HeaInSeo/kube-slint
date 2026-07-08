package gate_test

import (
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/gate"
)

// These exercise gate.CompareOp/LowerIsBetter/HigherIsBetter directly since
// they're now called by cmd/slint-gate (baseline diff/merge, recommend-policy)
// as public API, not just indirectly through Evaluate.

func TestCompareOp(t *testing.T) {
	cases := []struct {
		op   string
		v, t float64
		want bool
	}{
		{"<=", 1, 2, true},
		{"<=", 2, 2, true},
		{"<=", 3, 2, false},
		{">=", 3, 2, true},
		{"<", 1, 2, true},
		{">", 3, 2, true},
		{"==", 2, 2, true},
		{"==", 2, 3, false},
	}
	for _, c := range cases {
		got, err := gate.CompareOp(c.v, c.op, c.t)
		if err != nil {
			t.Fatalf("CompareOp(%v, %q, %v): unexpected error: %v", c.v, c.op, c.t, err)
		}
		if got != c.want {
			t.Errorf("CompareOp(%v, %q, %v) = %v, want %v", c.v, c.op, c.t, got, c.want)
		}
	}
}

func TestCompareOp_UnsupportedOperator(t *testing.T) {
	if _, err := gate.CompareOp(1, "~=", 2); err == nil {
		t.Fatal("expected an error for an unsupported operator")
	}
}

func TestLowerIsBetter(t *testing.T) {
	for _, op := range []string{"<=", "<", "=<"} {
		if !gate.LowerIsBetter(op) {
			t.Errorf("LowerIsBetter(%q) = false, want true", op)
		}
	}
	for _, op := range []string{">=", ">", "==", ""} {
		if gate.LowerIsBetter(op) {
			t.Errorf("LowerIsBetter(%q) = true, want false", op)
		}
	}
}

func TestHigherIsBetter(t *testing.T) {
	for _, op := range []string{">=", ">", "=>"} {
		if !gate.HigherIsBetter(op) {
			t.Errorf("HigherIsBetter(%q) = false, want true", op)
		}
	}
	for _, op := range []string{"<=", "<", "==", ""} {
		if gate.HigherIsBetter(op) {
			t.Errorf("HigherIsBetter(%q) = true, want false", op)
		}
	}
}
