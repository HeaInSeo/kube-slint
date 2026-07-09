package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunInit_GeneratesPolicyFile(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, ".slint", "policy.yaml")

	err := runInit([]string{"--output", out})
	require.NoError(t, err)

	data, err := os.ReadFile(out)
	require.NoError(t, err)

	body := string(data)
	assert.Contains(t, body, "schema_version")
	assert.Contains(t, body, "reconcile_total_delta")
	assert.Contains(t, body, "workqueue_depth_end")
	assert.Contains(t, body, "rest_client_429_delta")
}

func TestRunInit_CreatesSlintDir(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "nested", "sub", "policy.yaml")

	err := runInit([]string{"--output", out})
	require.NoError(t, err)
	assert.FileExists(t, out)
}

func TestRunInit_WithServiceOverride(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "policy.yaml")

	var err error
	stdout := captureStdout(t, func() {
		err = runInit([]string{
			"--output", out,
			"--service", "my-operator-metrics-service",
			"--namespace", "my-ns",
		})
	})

	require.NoError(t, err)
	assert.Contains(t, stdout, "my-operator-metrics-service")
	assert.Contains(t, stdout, "my-ns")
}

func TestRunInit_DefaultPlaceholders(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "policy.yaml")

	var err error
	stdout := captureStdout(t, func() {
		err = runInit([]string{"--output", out})
	})

	require.NoError(t, err)
	// 네임스페이스/서비스 미지정 시 placeholder가 출력되어야 함
	assert.Contains(t, stdout, "<YOUR_NAMESPACE>")
	assert.Contains(t, stdout, "<YOUR_METRICS_SERVICE_NAME>")
}

func TestResolveServiceName_Override(t *testing.T) {
	svc, candidates, err := resolveServiceName("ns", "my-svc")
	assert.Equal(t, "my-svc", svc)
	assert.Nil(t, candidates)
	assert.NoError(t, err)
}

func TestResolveServiceName_NoNamespace(t *testing.T) {
	svc, candidates, err := resolveServiceName("", "")
	assert.Empty(t, svc)
	assert.Nil(t, candidates)
	assert.NoError(t, err)
}

func TestIsMetricsServiceCandidate(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"my-operator-controller-manager-metrics-service", true},
		{"controller-manager", true},
		{"metrics-server", true},
		{"manager-service", true},
		{"random-svc", false},
		{"postgres", false},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, isMetricsServiceCandidate(tc.name), tc.name)
	}
}

func TestSnippetData_Placeholders(t *testing.T) {
	d := snippetData(initTemplateData{})
	assert.Equal(t, "<YOUR_NAMESPACE>", d.Namespace)
	assert.Equal(t, "<YOUR_METRICS_SERVICE_NAME>", d.ServiceName)
}

func TestSnippetData_PreservesValues(t *testing.T) {
	d := snippetData(initTemplateData{Namespace: "ns", ServiceName: "svc"})
	assert.Equal(t, "ns", d.Namespace)
	assert.Equal(t, "svc", d.ServiceName)
}

func TestRunInit_WithProfile_AddsProfileComment(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "policy.yaml")

	err := runInit([]string{"--output", out, "--profile", "kubebuilder-operator"})
	require.NoError(t, err)

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Contains(t, string(data), "# Profile:      kubebuilder-operator")
}

func TestRunInit_NoProfile_OmitsProfileComment(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "policy.yaml")

	err := runInit([]string{"--output", out})
	require.NoError(t, err)

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "# Profile:")
}

func TestRunInit_UnknownProfile_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "policy.yaml")

	err := runInit([]string{"--output", out, "--profile", "bogus"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown profile")
	assert.NoFileExists(t, out)
}

func TestRunInit_EmitRBAC(t *testing.T) {
	dir := t.TempDir()
	policyOut := filepath.Join(dir, "policy.yaml")
	rbacOut := filepath.Join(dir, "rbac-slint.yaml")

	err := runInit([]string{
		"--output", policyOut,
		"--namespace", "my-ns",
		"--emit-rbac", rbacOut,
	})
	require.NoError(t, err)

	require.FileExists(t, rbacOut)
	data, err := os.ReadFile(rbacOut)
	require.NoError(t, err)

	body := string(data)
	assert.Contains(t, body, "ServiceAccount")
	assert.Contains(t, body, "kind: Role")
	assert.Contains(t, body, "kind: RoleBinding")
	assert.NotContains(t, body, "ClusterRole")
	// kube-slint-no-clusterrolebinding-default: this asserts the string's
	// absence, not its presence.
	// nosemgrep
	assert.NotContains(t, body, "ClusterRoleBinding")
	assert.Contains(t, body, "my-ns")
	assert.Contains(t, body, "kube-slint-scraper")
}

func TestRunInit_EmitRBAC_NoNamespace(t *testing.T) {
	dir := t.TempDir()
	policyOut := filepath.Join(dir, "policy.yaml")
	rbacOut := filepath.Join(dir, "rbac.yaml")

	// No --namespace: namespace placeholder should appear in RBAC
	err := runInit([]string{"--output", policyOut, "--emit-rbac", rbacOut})
	require.NoError(t, err)

	data, _ := os.ReadFile(rbacOut)
	assert.Contains(t, string(data), "kind: RoleBinding")
	assert.Contains(t, string(data), "<YOUR_NAMESPACE>")
}

func TestRunInit_RefusesOverwriteWithoutForce(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "policy.yaml")
	require.NoError(t, os.WriteFile(out, []byte("existing content"), 0o644))

	err := runInit([]string{"--output", out})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	data, readErr := os.ReadFile(out)
	require.NoError(t, readErr)
	assert.Equal(t, "existing content", string(data), "existing file must not be touched without --force")
}

func TestRunInit_ForceOverwrites(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "policy.yaml")
	require.NoError(t, os.WriteFile(out, []byte("existing content"), 0o644))

	err := runInit([]string{"--output", out, "--force"})
	require.NoError(t, err)

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Contains(t, string(data), "schema_version")
}

func TestRunInit_RefusesRBACOverwriteWithoutForce(t *testing.T) {
	dir := t.TempDir()
	policyOut := filepath.Join(dir, "policy.yaml")
	rbacOut := filepath.Join(dir, "rbac.yaml")
	require.NoError(t, os.WriteFile(rbacOut, []byte("existing rbac"), 0o644))

	err := runInit([]string{"--output", policyOut, "--emit-rbac", rbacOut})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	assert.NoFileExists(t, policyOut, "policy.yaml must not be written when the RBAC overwrite check fails")

	data, readErr := os.ReadFile(rbacOut)
	require.NoError(t, readErr)
	assert.Equal(t, "existing rbac", string(data), "existing RBAC file must not be touched without --force")
}

func TestRunInit_ForceOverwritesRBAC(t *testing.T) {
	dir := t.TempDir()
	policyOut := filepath.Join(dir, "policy.yaml")
	rbacOut := filepath.Join(dir, "rbac.yaml")
	require.NoError(t, os.WriteFile(rbacOut, []byte("existing rbac"), 0o644))

	err := runInit([]string{"--output", policyOut, "--emit-rbac", rbacOut, "--force"})
	require.NoError(t, err)

	data, err := os.ReadFile(rbacOut)
	require.NoError(t, err)
	assert.Contains(t, string(data), "ServiceAccount")
}

func TestDescribeKubectlError_ExitErrorSurfacesStderr(t *testing.T) {
	cmd := exec.Command("sh", "-c", "echo 'boom: no such namespace' >&2; exit 1")
	_, err := cmd.Output()
	require.Error(t, err)

	got := describeKubectlError(err)
	assert.Contains(t, got.Error(), "boom: no such namespace")
}

func TestDescribeKubectlError_NonExitErrorSurfacesGoError(t *testing.T) {
	cmd := exec.Command("slint-gate-definitely-not-a-real-binary")
	_, err := cmd.Output()
	require.Error(t, err)

	got := describeKubectlError(err)
	assert.Contains(t, got.Error(), "kubectl get svc")
}

func TestPrintDiscoveryResult_DiscoverErrorIsDistinctFromNoCandidates(t *testing.T) {
	errOutput := captureStdout(t, func() {
		printDiscoveryResult("my-ns", nil, errors.New("kubectl get svc: connection refused"))
	})
	assert.Contains(t, errOutput, "could not auto-detect")
	assert.Contains(t, errOutput, "connection refused")

	noCandidatesOutput := captureStdout(t, func() {
		printDiscoveryResult("my-ns", nil, nil)
	})
	assert.Contains(t, noCandidatesOutput, "no metrics services auto-detected")
	assert.NotContains(t, noCandidatesOutput, "could not auto-detect")
}
