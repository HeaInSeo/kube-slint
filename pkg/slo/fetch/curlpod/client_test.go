package curlpod

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

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

// security/privileged-curlpod.yaml, security/hostpath-curlpod.yaml
func TestRunOnce_PodOverrides_NeverPrivilegedOrHostPath(t *testing.T) {
	r := &argsCaptureRunner{}
	c := New(nil, r)

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
	if strings.Contains(overridesJSON, "hostPath") {
		t.Fatalf("pod override must never contain a hostPath volume: %s", overridesJSON)
	}
	if strings.Contains(overridesJSON, `"privileged": true`) || strings.Contains(overridesJSON, `"privileged":true`) {
		t.Fatalf("pod override must never set privileged: true: %s", overridesJSON)
	}

	var overrides struct {
		Spec struct {
			Containers []struct {
				SecurityContext struct {
					AllowPrivilegeEscalation bool `json:"allowPrivilegeEscalation"`
					RunAsNonRoot             bool `json:"runAsNonRoot"`
					Capabilities             struct {
						Drop []string `json:"drop"`
					} `json:"capabilities"`
					SeccompProfile struct {
						Type string `json:"type"`
					} `json:"seccompProfile"`
				} `json:"securityContext"`
			} `json:"containers"`
		} `json:"spec"`
	}
	if err := json.Unmarshal([]byte(overridesJSON), &overrides); err != nil {
		t.Fatalf("--overrides is not valid JSON: %v\n%s", err, overridesJSON)
	}
	require1 := overrides.Spec.Containers
	if len(require1) != 1 {
		t.Fatalf("expected exactly one container, got %d", len(require1))
	}
	sc := require1[0].SecurityContext
	if sc.AllowPrivilegeEscalation {
		t.Fatal("expected allowPrivilegeEscalation=false")
	}
	if !sc.RunAsNonRoot {
		t.Fatal("expected runAsNonRoot=true")
	}
	if len(sc.Capabilities.Drop) != 1 || sc.Capabilities.Drop[0] != "ALL" {
		t.Fatalf("expected capabilities.drop=[ALL], got %v", sc.Capabilities.Drop)
	}
	if sc.SeccompProfile.Type != "RuntimeDefault" {
		t.Fatalf("expected seccompProfile.type=RuntimeDefault, got %q", sc.SeccompProfile.Type)
	}
}

// TestRunOnce_RejectsServiceAccountNamePodSpecInjection reproduces the
// finding from the second pre-release-adversarial-review pass (2026-07-08):
// a crafted ServiceAccount name used to be spliced unescaped into a
// hand-built --overrides JSON string via fmt.Sprintf, letting an attacker
// smuggle extra PodSpec fields (hostNetwork, hostPath volumes, a privileged
// initContainer) past the "never privileged / never hostPath" invariant.
// RunOnce now validates serviceAccountName as a DNS-1123 label before it
// ever reaches JSON construction, so this must be rejected outright.
func TestRunOnce_RejectsServiceAccountNamePodSpecInjection(t *testing.T) {
	r := &argsCaptureRunner{}
	c := New(nil, r)

	malicious := `x","hostNetwork":true,"hostPID":true,"volumes":[{"name":"host","hostPath":{"path":"/"}}],"initContainers":[{"name":"evil","image":"alpine","command":["sh","-c","sleep 3600"],"securityContext":{"privileged":true},"volumeMounts":[{"name":"host","mountPath":"/host"}]}],"z":"z`

	_, err := c.RunOnce(context.Background(), "ns", "", "metrics-svc", malicious)
	if err == nil {
		t.Fatal("expected RunOnce to reject an invalid ServiceAccount name, got nil error")
	}
	if !strings.Contains(err.Error(), "invalid ServiceAccount name") {
		t.Fatalf("expected an 'invalid ServiceAccount name' error, got: %v", err)
	}
	if len(r.args) != 0 {
		t.Fatalf("kubectl must never be invoked for a rejected ServiceAccount name, got args: %v", r.args)
	}
}

// TestPodOverrideMarshal_ServiceAccountNameCannotBreakOutOfJSON verifies the
// structural fix independent of the DNS-label validation guard above: even
// a string containing raw quotes/braces cannot inject sibling JSON keys once
// the --overrides payload is built via encoding/json.Marshal of a typed
// struct instead of fmt.Sprintf string interpolation. This is the same
// property the pre-fix code was missing for ServiceAccountName and Image.
func TestPodOverrideMarshal_ServiceAccountNameCannotBreakOutOfJSON(t *testing.T) {
	malicious := `x","hostNetwork":true,"z":"z`

	data, err := json.Marshal(podOverride{
		APIVersion: "v1",
		Kind:       "Pod",
		Metadata:   podOverrideMetadata{Name: "p", Namespace: "ns", Labels: map[string]string{"app": "curl-metrics"}},
		Spec: podOverrideSpec{
			ServiceAccountName: malicious,
			RestartPolicy:      "Never",
			Containers:         []podOverrideContainer{{Name: "curl", Image: malicious}},
		},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded struct {
		Spec struct {
			ServiceAccountName string `json:"serviceAccountName"`
			HostNetwork        *bool  `json:"hostNetwork"`
			Containers         []struct {
				Image string `json:"image"`
			} `json:"containers"`
		} `json:"spec"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("--overrides is not valid JSON: %v\n%s", err, data)
	}
	if decoded.Spec.HostNetwork != nil {
		t.Fatalf("malicious ServiceAccountName must not be able to inject a sibling hostNetwork field: %s", data)
	}
	if decoded.Spec.ServiceAccountName != malicious {
		t.Fatalf("serviceAccountName must round-trip as a literal string, got %q", decoded.Spec.ServiceAccountName)
	}
	if len(decoded.Spec.Containers) != 1 || decoded.Spec.Containers[0].Image != malicious {
		t.Fatalf("image must round-trip as a literal string: %s", data)
	}
}

// security/kube-system-target.yaml
func TestRunOnce_RejectsKubeSystemNamespaceByDefault(t *testing.T) {
	r := &argsCaptureRunner{}
	c := New(nil, r)

	_, err := c.RunOnce(context.Background(), "kube-system", "", "metrics-svc", "scraper-sa")
	if err == nil {
		t.Fatal("expected kube-system namespace to be rejected by default")
	}
	if len(r.args) != 0 {
		t.Fatalf("expected no kubectl command to run before namespace validation, got: %v", r.args)
	}
}

func TestRunOnce_DangerouslyAllowKubeSystemNamespace_Opt(t *testing.T) {
	r := &argsCaptureRunner{}
	c := New(nil, r)
	c.DangerouslyAllowKubeSystemNamespace = true

	_, err := c.RunOnce(context.Background(), "kube-system", "", "metrics-svc", "scraper-sa")
	if err != nil {
		t.Fatalf("expected explicit opt-in to allow kube-system namespace, got error: %v", err)
	}
}

// security/external-service-url.yaml — enforced at RunOnce, before any kubectl command
func TestRunOnce_RejectsExternalServiceURLFormatByDefault(t *testing.T) {
	r := &argsCaptureRunner{}
	c := New(nil, r)
	c.ServiceURLFormat = "https://evil.example.com/collect?svc=%s&ns=%s"

	_, err := c.RunOnce(context.Background(), "ns", "", "metrics-svc", "scraper-sa")
	if err == nil {
		t.Fatal("expected external ServiceURLFormat to be rejected by default")
	}
	if len(r.args) != 0 {
		t.Fatalf("expected no kubectl command to run before URL validation, got: %v", r.args)
	}
}

func TestRunOnce_DangerouslyAllowExternalMetricsURL_Opt(t *testing.T) {
	r := &argsCaptureRunner{}
	c := New(nil, r)
	c.ServiceURLFormat = "https://evil.example.com/collect?svc=%s&ns=%s"
	c.DangerouslyAllowExternalMetricsURL = true

	_, err := c.RunOnce(context.Background(), "ns", "", "metrics-svc", "scraper-sa")
	if err != nil {
		t.Fatalf("expected explicit opt-in to allow external ServiceURLFormat, got error: %v", err)
	}
}

func TestNew_TLSInsecureSkipVerifyDefaultsFalse(t *testing.T) {
	c := New(nil, nil)
	if c.TLSInsecureSkipVerify {
		t.Fatal("expected TLSInsecureSkipVerify to default to false")
	}
	if c.DangerouslySkipTLSVerify {
		t.Fatal("expected DangerouslySkipTLSVerify to default to false")
	}
}

// phaseRunner returns a scripted phase for "kubectl get pod ... jsonpath=
// {.status.phase}" and nil/empty for anything else (e.g. delete cleanup).
type phaseRunner struct {
	phase    string
	phaseErr error
}

func (r *phaseRunner) Run(_ context.Context, _ slo.Logger, cmd *exec.Cmd) (string, error) {
	for _, a := range cmd.Args {
		if a == "jsonpath={.status.phase}" {
			return r.phase, r.phaseErr
		}
	}
	return "", nil
}

func TestWaitDone_Succeeded_ReturnsNil(t *testing.T) {
	c := New(nil, &phaseRunner{phase: "Succeeded"})
	if err := c.WaitDone(context.Background(), "ns", "pod-1", time.Millisecond); err != nil {
		t.Fatalf("expected nil error for phase=Succeeded, got: %v", err)
	}
}

// TestWaitDone_Failed_ReturnsErrPodFailed reproduces the finding from the
// third pre-release-adversarial-review pass (2026-07-09): WaitDone used to
// treat phase=Succeeded and phase=Failed identically (both just "done"),
// so a genuinely failed scrape was silently treated as a successful one
// downstream. WaitDone must now distinguish them.
func TestWaitDone_Failed_ReturnsErrPodFailed(t *testing.T) {
	c := New(nil, &phaseRunner{phase: "Failed"})
	err := c.WaitDone(context.Background(), "ns", "pod-1", time.Millisecond)
	if err == nil {
		t.Fatal("expected an error for phase=Failed, got nil")
	}
	if !errors.Is(err, ErrPodFailed) {
		t.Fatalf("expected errors.Is(err, ErrPodFailed) to be true, got: %v", err)
	}
	if !strings.Contains(err.Error(), "ns/pod-1") {
		t.Fatalf("expected the error to identify the pod, got: %v", err)
	}
}

func TestWaitDone_KubectlError_PropagatesWithoutErrPodFailed(t *testing.T) {
	c := New(nil, &phaseRunner{phaseErr: context.DeadlineExceeded})
	err := c.WaitDone(context.Background(), "ns", "pod-1", time.Millisecond)
	if err == nil {
		t.Fatal("expected an error when the phase check itself fails, got nil")
	}
	if errors.Is(err, ErrPodFailed) {
		t.Fatal("a kubectl/context error must not be reported as ErrPodFailed")
	}
}
