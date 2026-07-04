package service_test

import (
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/dataplane"
	"github.com/stretchr/testify/assert"
)

func TestResourcesCheck_FullySet_Pass(t *testing.T) {
	b := &dataplane.Bundle{Workloads: []dataplane.Workload{
		workload(dataplane.Container{
			Name: "c",
			Resources: dataplane.ResourceRequirements{
				Requests: dataplane.ResourceList{"cpu": "10m", "memory": "32Mi"},
				Limits:   dataplane.ResourceList{"cpu": "200m", "memory": "128Mi"},
			},
		}),
	}}
	assert.Empty(t, runCheck(t, "KSL-DP-005", b))
}

func TestResourcesCheck_LimitsMissingMemory_OneFinding(t *testing.T) {
	b := &dataplane.Bundle{Workloads: []dataplane.Workload{
		workload(dataplane.Container{
			Name: "c",
			Resources: dataplane.ResourceRequirements{
				Requests: dataplane.ResourceList{"cpu": "10m", "memory": "32Mi"},
				Limits:   dataplane.ResourceList{"cpu": "200m"},
			},
		}),
	}}
	msgs := runCheck(t, "KSL-DP-005", b)
	assert.Len(t, msgs, 1)
	assert.Contains(t, msgs[0], "limits")
	assert.Contains(t, msgs[0], "memory")
}

func TestResourcesCheck_RequestsBlockAbsent_NamesBothKeys(t *testing.T) {
	b := &dataplane.Bundle{Workloads: []dataplane.Workload{
		workload(dataplane.Container{
			Name: "c",
			Resources: dataplane.ResourceRequirements{
				Limits: dataplane.ResourceList{"cpu": "200m", "memory": "128Mi"},
			},
		}),
	}}
	msgs := runCheck(t, "KSL-DP-005", b)
	assert.Len(t, msgs, 1)
	assert.Contains(t, msgs[0], "requests")
	assert.Contains(t, msgs[0], "cpu")
	assert.Contains(t, msgs[0], "memory")
}

func TestResourcesCheck_BothBlocksMissing_TwoFindings(t *testing.T) {
	b := &dataplane.Bundle{Workloads: []dataplane.Workload{
		workload(dataplane.Container{Name: "c"}),
	}}
	assert.Len(t, runCheck(t, "KSL-DP-005", b), 2)
}
