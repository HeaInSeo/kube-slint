package spec

import "testing"

func TestJUMIAHMinimumSpecs(t *testing.T) {
	specs := JUMIAHMinimumSpecs()
	if len(specs) < 6 {
		t.Fatalf("len(specs) = %d, want at least 6", len(specs))
	}
	seen := map[string]bool{}
	for _, s := range specs {
		if s.ID == "" {
			t.Fatal("spec id must not be empty")
		}
		if len(s.Inputs) == 0 {
			t.Fatalf("spec %s must have at least one input", s.ID)
		}
		if seen[s.ID] {
			t.Fatalf("duplicate spec id: %s", s.ID)
		}
		seen[s.ID] = true
	}
	if !seen["jumi_jobs_created_delta"] {
		t.Fatal("expected jumi_jobs_created_delta")
	}
	if !seen["ah_resolve_requests_delta"] {
		t.Fatal("expected ah_resolve_requests_delta")
	}
}
