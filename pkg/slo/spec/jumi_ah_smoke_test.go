package spec

import "testing"

func TestJUMIAHSmokeGuardrailSpecs(t *testing.T) {
	specs := JUMIAHSmokeGuardrailSpecs()
	if len(specs) < 8 {
		t.Fatalf("len(specs) = %d, want at least 8", len(specs))
	}
	seen := map[string]bool{}
	for _, s := range specs {
		if s.ID == "" {
			t.Fatal("spec id must not be empty")
		}
		if len(s.Inputs) == 0 {
			t.Fatalf("spec %s must have at least one input", s.ID)
		}
		if s.Judge == nil || len(s.Judge.Rules) == 0 {
			t.Fatalf("spec %s must define at least one smoke guardrail rule", s.ID)
		}
		if seen[s.ID] {
			t.Fatalf("duplicate spec id: %s", s.ID)
		}
		seen[s.ID] = true
	}
	if !seen["jumi_input_resolve_requests_smoke"] {
		t.Fatal("expected jumi_input_resolve_requests_smoke")
	}
	if !seen["ah_gc_backlog_bytes_smoke"] {
		t.Fatal("expected ah_gc_backlog_bytes_smoke")
	}
}
