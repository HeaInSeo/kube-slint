package service_test

import (
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/dataplane"
	"github.com/HeaInSeo/kube-slint/pkg/dataplane/service"
	"github.com/HeaInSeo/kube-slint/pkg/report"
	"github.com/stretchr/testify/assert"
)

func noopCheck(id string) service.CheckDef {
	return service.CheckDef{ID: id, Run: func(*dataplane.Bundle) []report.Finding { return nil }}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := service.NewRegistry()
	assert.NoError(t, r.Register(noopCheck("KSL-DP-999")))

	c, ok := r.Get("KSL-DP-999")
	assert.True(t, ok)
	assert.Equal(t, "KSL-DP-999", c.ID)

	_, ok = r.Get("does-not-exist")
	assert.False(t, ok)
}

func TestRegistry_DuplicateIDIsError(t *testing.T) {
	r := service.NewRegistry()
	assert.NoError(t, r.Register(noopCheck("KSL-DP-999")))
	assert.Error(t, r.Register(noopCheck("KSL-DP-999")))
}

func TestRegistry_EmptyIDIsError(t *testing.T) {
	r := service.NewRegistry()
	assert.Error(t, r.Register(noopCheck("")))
}

func TestRegistry_MustRegisterPanicsOnDuplicate(t *testing.T) {
	r := service.NewRegistry()
	r.MustRegister(noopCheck("KSL-DP-999"))
	assert.Panics(t, func() { r.MustRegister(noopCheck("KSL-DP-999")) })
}

func TestRegistry_ListIsSortedByID(t *testing.T) {
	r := service.NewRegistry()
	r.MustRegister(noopCheck("KSL-DP-003"))
	r.MustRegister(noopCheck("KSL-DP-001"))
	r.MustRegister(noopCheck("KSL-DP-002"))

	list := r.List()
	assert.Equal(t, []string{"KSL-DP-001", "KSL-DP-002", "KSL-DP-003"}, []string{list[0].ID, list[1].ID, list[2].ID})
}

func TestDefaultRegistry_HasAllSixChecks(t *testing.T) {
	r := service.DefaultRegistry()
	list := r.List()
	require := []string{"KSL-DP-001", "KSL-DP-002", "KSL-DP-003", "KSL-DP-004", "KSL-DP-005", "KSL-DP-006"}
	got := make([]string, len(list))
	for i, c := range list {
		got[i] = c.ID
	}
	assert.Equal(t, require, got)
}
