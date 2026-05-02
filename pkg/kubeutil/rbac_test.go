package kubeutil

import (
	"context"
	"io"
	"os/exec"
	"strings"
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/slo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// captureRunner records the stdin that would be piped to kubectl.
type captureRunner struct {
	stdin  string
	stdout string
	err    error
}

func (c *captureRunner) Run(_ context.Context, _ slo.Logger, cmd *exec.Cmd) (string, error) {
	if cmd.Stdin != nil {
		b, _ := io.ReadAll(cmd.Stdin)
		c.stdin = string(b)
	}
	return c.stdout, c.err
}

func TestApplyClusterRoleBinding_ValidYAML(t *testing.T) {
	r := &captureRunner{}
	err := ApplyClusterRoleBinding(context.Background(), nil, r,
		"my-crb", "my-cluster-role", "my-ns", "my-sa")
	require.NoError(t, err)

	var doc clusterRoleBindingDoc
	require.NoError(t, yaml.Unmarshal([]byte(r.stdin), &doc))

	assert.Equal(t, "rbac.authorization.k8s.io/v1", doc.APIVersion)
	assert.Equal(t, "ClusterRoleBinding", doc.Kind)
	assert.Equal(t, "my-crb", doc.Metadata.Name)
	assert.Equal(t, "my-cluster-role", doc.RoleRef.Name)
	assert.Equal(t, "ClusterRole", doc.RoleRef.Kind)
	assert.Equal(t, "rbac.authorization.k8s.io", doc.RoleRef.APIGroup)
	require.Len(t, doc.Subjects, 1)
	assert.Equal(t, "ServiceAccount", doc.Subjects[0].Kind)
	assert.Equal(t, "my-sa", doc.Subjects[0].Name)
	assert.Equal(t, "my-ns", doc.Subjects[0].Namespace)
}

// Values containing YAML special characters must be safely escaped, not injected.
func TestApplyClusterRoleBinding_YAMLSpecialChars(t *testing.T) {
	r := &captureRunner{}
	err := ApplyClusterRoleBinding(context.Background(), nil, r,
		"crb: injected\nkind: Evil", "role", "ns", "sa")
	require.NoError(t, err)

	var doc clusterRoleBindingDoc
	require.NoError(t, yaml.Unmarshal([]byte(r.stdin), &doc))

	// The injected string must appear as the literal metadata name, not break structure.
	assert.Equal(t, "crb: injected\nkind: Evil", doc.Metadata.Name)
	assert.Equal(t, "ClusterRoleBinding", doc.Kind)
}

func TestApplyClusterRoleBinding_CommandArguments(t *testing.T) {
	r := &captureRunner{}
	_ = ApplyClusterRoleBinding(context.Background(), nil, r,
		"crb", "role", "ns", "sa")

	// The manifest must be YAML parseable (not raw fmt.Sprintf injection).
	assert.True(t, strings.HasPrefix(r.stdin, "apiVersion:"),
		"manifest should start with apiVersion field")
}

func TestApplyClusterRoleBinding_RunnerError(t *testing.T) {
	r := &captureRunner{err: assert.AnError}
	err := ApplyClusterRoleBinding(context.Background(), nil, r,
		"crb", "role", "ns", "sa")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "kubectl apply clusterrolebinding failed")
}
