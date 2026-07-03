package curlpod

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/slo"
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

type captureRunner struct {
	commands []string
}

func (r *captureRunner) Run(_ context.Context, _ slo.Logger, cmd *exec.Cmd) (string, error) {
	r.commands = append(r.commands, strings.Join(cmd.Args, " "))
	return "", nil
}

func TestRunOnce_DoesNotEmbedProvidedTokenInCommand(t *testing.T) {
	r := &captureRunner{}
	c := New(nil, r)
	c.Image = "curlimages/curl:8.11.0"

	_, err := c.RunOnce(context.Background(), "ns", "super-secret-token", "metrics-svc", "scraper")
	if err != nil {
		t.Fatalf("RunOnce returned error: %v", err)
	}

	if len(r.commands) == 0 {
		t.Fatal("expected captured kubectl commands")
	}
	all := strings.Join(r.commands, "\n")
	if strings.Contains(all, "super-secret-token") {
		t.Fatalf("provided token leaked into command args: %s", all)
	}
	if !strings.Contains(all, "/var/run/secrets/kubernetes.io/serviceaccount/token") {
		t.Fatalf("expected pod command to read mounted service account token: %s", all)
	}
}

type argsCaptureRunner struct {
	args []string
}

func (r *argsCaptureRunner) Run(_ context.Context, _ slo.Logger, cmd *exec.Cmd) (string, error) {
	r.args = cmd.Args
	return "", nil
}

func TestRunOnce_PodOverrides_SetsAutomountServiceAccountTokenAndIsValidJSON(t *testing.T) {
	// Regression test for N2: since the pod reads its own mounted SA token
	// instead of a caller-supplied token, automountServiceAccountToken must be
	// explicitly requested rather than relying on the ServiceAccount's default
	// (which may have automount disabled).
	r := &argsCaptureRunner{}
	c := New(nil, r)
	c.Image = "curlimages/curl:8.11.0"

	_, err := c.RunOnce(context.Background(), "ns", "", "metrics-svc", "scraper-sa")
	if err != nil {
		t.Fatalf("RunOnce returned error: %v", err)
	}

	var overridesJSON string
	for i, a := range r.args {
		if a == "--overrides" && i+1 < len(r.args) {
			overridesJSON = r.args[i+1]
			break
		}
	}
	if overridesJSON == "" {
		t.Fatalf("expected --overrides argument, got args: %v", r.args)
	}

	var overrides struct {
		Spec struct {
			ServiceAccountName           string `json:"serviceAccountName"`
			AutomountServiceAccountToken bool   `json:"automountServiceAccountToken"`
		} `json:"spec"`
	}
	if err := json.Unmarshal([]byte(overridesJSON), &overrides); err != nil {
		t.Fatalf("--overrides is not valid JSON: %v\n%s", err, overridesJSON)
	}
	if overrides.Spec.ServiceAccountName != "scraper-sa" {
		t.Fatalf("unexpected serviceAccountName: %q", overrides.Spec.ServiceAccountName)
	}
	if !overrides.Spec.AutomountServiceAccountToken {
		t.Fatal("expected automountServiceAccountToken=true so the pod can read its own SA token")
	}
}
