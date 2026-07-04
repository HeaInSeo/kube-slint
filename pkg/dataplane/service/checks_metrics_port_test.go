package service_test

import (
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/dataplane"
	"github.com/HeaInSeo/kube-slint/pkg/dataplane/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func workload(containers ...dataplane.Container) dataplane.Workload {
	return dataplane.Workload{
		Kind:       "Deployment",
		Metadata:   dataplane.ObjectMeta{Name: "app", Namespace: "ns"},
		SourceFile: "app.yaml",
		Spec: dataplane.WorkloadSpec{
			Template: dataplane.PodTemplateSpec{
				Metadata: dataplane.ObjectMeta{Labels: map[string]string{"app": "app"}},
				Spec:     dataplane.PodSpec{Containers: containers},
			},
		},
	}
}

func runCheck(t *testing.T, id string, b *dataplane.Bundle) []string {
	t.Helper()
	reg := service.DefaultRegistry()
	c, ok := reg.Get(id)
	require.True(t, ok)
	findings := c.Run(b)
	msgs := make([]string, len(findings))
	for i, f := range findings {
		assert.Equal(t, id, f.RuleID)
		msgs[i] = f.Message
	}
	return msgs
}

func TestMetricsPortCheck_HasMetricsPort_Pass(t *testing.T) {
	b := &dataplane.Bundle{Workloads: []dataplane.Workload{
		workload(dataplane.Container{Name: "c", Ports: []dataplane.ContainerPort{{Name: "metrics", ContainerPort: 8080}}}),
	}}
	assert.Empty(t, runCheck(t, "KSL-DP-001", b))
}

func TestMetricsPortCheck_CaseInsensitive_Pass(t *testing.T) {
	b := &dataplane.Bundle{Workloads: []dataplane.Workload{
		workload(dataplane.Container{Name: "c", Ports: []dataplane.ContainerPort{{Name: "Metrics", ContainerPort: 8080}}}),
	}}
	assert.Empty(t, runCheck(t, "KSL-DP-001", b))
}

func TestMetricsPortCheck_NoMetricsPort_Fail(t *testing.T) {
	b := &dataplane.Bundle{Workloads: []dataplane.Workload{
		workload(dataplane.Container{Name: "c", Ports: []dataplane.ContainerPort{{Name: "http", ContainerPort: 80}}}),
	}}
	assert.Len(t, runCheck(t, "KSL-DP-001", b), 1)
}

func TestMetricsPortCheck_SidecarPattern_OnlyOneContainerHasPort_Pass(t *testing.T) {
	b := &dataplane.Bundle{Workloads: []dataplane.Workload{
		workload(
			dataplane.Container{Name: "app", Ports: []dataplane.ContainerPort{{Name: "http", ContainerPort: 80}}},
			dataplane.Container{Name: "sidecar", Ports: []dataplane.ContainerPort{{Name: "metrics", ContainerPort: 9090}}},
		),
	}}
	assert.Empty(t, runCheck(t, "KSL-DP-001", b))
}
