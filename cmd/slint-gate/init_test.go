package main

import (
	"os"
	"path/filepath"
	"strings"
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

	// Capture stdout via pipe
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runInit([]string{
		"--output", out,
		"--service", "my-operator-metrics-service",
		"--namespace", "my-ns",
	})

	_ = w.Close()
	os.Stdout = old

	var buf strings.Builder
	tmp := make([]byte, 4096)
	for {
		n, _ := r.Read(tmp)
		if n == 0 {
			break
		}
		buf.Write(tmp[:n])
	}

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "my-operator-metrics-service")
	assert.Contains(t, buf.String(), "my-ns")
}

func TestRunInit_DefaultPlaceholders(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "policy.yaml")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runInit([]string{"--output", out})

	_ = w.Close()
	os.Stdout = old

	var buf strings.Builder
	tmp := make([]byte, 4096)
	for {
		n, _ := r.Read(tmp)
		if n == 0 {
			break
		}
		buf.Write(tmp[:n])
	}

	require.NoError(t, err)
	// 네임스페이스/서비스 미지정 시 placeholder가 출력되어야 함
	assert.Contains(t, buf.String(), "<YOUR_NAMESPACE>")
	assert.Contains(t, buf.String(), "<YOUR_METRICS_SERVICE_NAME>")
}

func TestResolveServiceName_Override(t *testing.T) {
	svc, candidates := resolveServiceName("ns", "my-svc")
	assert.Equal(t, "my-svc", svc)
	assert.Nil(t, candidates)
}

func TestResolveServiceName_NoNamespace(t *testing.T) {
	svc, candidates := resolveServiceName("", "")
	assert.Empty(t, svc)
	assert.Nil(t, candidates)
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
	assert.Contains(t, body, "ClusterRole")
	assert.Contains(t, body, "ClusterRoleBinding")
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
	assert.Contains(t, string(data), "ClusterRoleBinding")
}
