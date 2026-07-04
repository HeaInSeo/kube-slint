package service_test

import (
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/dataplane"
	"github.com/stretchr/testify/assert"
)

func workloadWithGracePeriod(seconds *int64) dataplane.Workload {
	w := workload(dataplane.Container{Name: "c"})
	w.Spec.Template.Spec.TerminationGracePeriodSeconds = seconds
	return w
}

func int64Ptr(v int64) *int64 { return &v }

func TestGracePeriodCheck_Unset_Warn(t *testing.T) {
	b := &dataplane.Bundle{Workloads: []dataplane.Workload{workloadWithGracePeriod(nil)}}
	assert.Len(t, runCheck(t, "KSL-DP-006", b), 1)
}

func TestGracePeriodCheck_ExplicitZero_Pass(t *testing.T) {
	b := &dataplane.Bundle{Workloads: []dataplane.Workload{workloadWithGracePeriod(int64Ptr(0))}}
	assert.Empty(t, runCheck(t, "KSL-DP-006", b))
}

func TestGracePeriodCheck_ExplicitValue_Pass(t *testing.T) {
	b := &dataplane.Bundle{Workloads: []dataplane.Workload{workloadWithGracePeriod(int64Ptr(30))}}
	assert.Empty(t, runCheck(t, "KSL-DP-006", b))
}
