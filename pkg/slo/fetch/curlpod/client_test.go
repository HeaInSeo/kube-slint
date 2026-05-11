package curlpod

import (
	"encoding/json"
	"testing"
)

func TestParseSelectorToLabels(t *testing.T) {
	cases := []struct {
		name     string
		selector string
		want     map[string]string
	}{
		{
			name:     "single pair",
			selector: "app=curl-metrics",
			want:     map[string]string{"app": "curl-metrics"},
		},
		{
			name:     "multi pair with namespaced key",
			selector: "app.kubernetes.io/managed-by=kube-slint,slint-run-id=abc123",
			want: map[string]string{
				"app.kubernetes.io/managed-by": "kube-slint",
				"slint-run-id":                 "abc123",
			},
		},
		{
			name:     "empty selector",
			selector: "",
			want:     map[string]string{},
		},
		{
			name:     "entry without equals is skipped",
			selector: "app=foo,no-equals,bar=baz",
			want:     map[string]string{"app": "foo", "bar": "baz"},
		},
		{
			name:     "spaces around pairs are trimmed",
			selector: " app = foo , bar = baz ",
			want:     map[string]string{"app": "foo", "bar": "baz"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseSelectorToLabels(tc.selector)
			if len(got) != len(tc.want) {
				t.Fatalf("len mismatch: got %v, want %v", got, tc.want)
			}
			for k, v := range tc.want {
				if got[k] != v {
					t.Errorf("key %q: got %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

// TestParseSelectorToLabels_JSONSafe verifies that labels built from the
// harness selector (with namespaced keys) are safe to embed in the override JSON.
func TestParseSelectorToLabels_JSONSafe(t *testing.T) {
	selector := "app.kubernetes.io/managed-by=kube-slint,slint-run-id=run-001"
	labels := parseSelectorToLabels(selector)
	labels["app"] = "curl-metrics" // mimic RunOnce fallback

	b, err := json.Marshal(labels)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var out map[string]string
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if out["app.kubernetes.io/managed-by"] != "kube-slint" {
		t.Errorf("unexpected value for managed-by: %q", out["app.kubernetes.io/managed-by"])
	}
	if out["slint-run-id"] != "run-001" {
		t.Errorf("unexpected value for slint-run-id: %q", out["slint-run-id"])
	}
	if out["app"] != "curl-metrics" {
		t.Errorf("unexpected value for app: %q", out["app"])
	}
}
