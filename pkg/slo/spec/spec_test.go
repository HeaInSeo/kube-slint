package spec_test

import (
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/slo/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Registry ---

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := spec.NewRegistry()
	s := spec.SLISpec{ID: "my_sli", Title: "My SLI"}
	require.NoError(t, r.Register(s))

	got, ok := r.Get("my_sli")
	assert.True(t, ok)
	assert.Equal(t, "My SLI", got.Title)
}

func TestRegistry_Get_NotFound(t *testing.T) {
	r := spec.NewRegistry()
	_, ok := r.Get("nonexistent")
	assert.False(t, ok)
}

func TestRegistry_Register_EmptyID(t *testing.T) {
	r := spec.NewRegistry()
	err := r.Register(spec.SLISpec{})
	assert.ErrorContains(t, err, "id is required")
}

func TestRegistry_Register_Duplicate(t *testing.T) {
	r := spec.NewRegistry()
	s := spec.SLISpec{ID: "dup"}
	require.NoError(t, r.Register(s))
	err := r.Register(s)
	assert.ErrorContains(t, err, "already registered")
}

func TestRegistry_MustRegister_Panics(t *testing.T) {
	r := spec.NewRegistry()
	require.NoError(t, r.Register(spec.SLISpec{ID: "x"}))
	assert.Panics(t, func() {
		r.MustRegister(spec.SLISpec{ID: "x"})
	})
}

func TestRegistry_List(t *testing.T) {
	r := spec.NewRegistry()
	r.MustRegister(spec.SLISpec{ID: "a"})
	r.MustRegister(spec.SLISpec{ID: "b"})
	list := r.List()
	assert.Len(t, list, 2)
}

// --- NormalizeOp ---

func TestNormalizeOp(t *testing.T) {
	cases := []struct {
		input string
		want  spec.Op
		ok    bool
	}{
		{"<=", spec.OpLE, true},
		{"=<", spec.OpLE, true},
		{">=", spec.OpGE, true},
		{"=>", spec.OpGE, true},
		{"<", spec.OpLT, true},
		{">", spec.OpGT, true},
		{"==", spec.OpEQ, true},
		{"=", spec.OpEQ, true},
		{"le", spec.OpLE, true},
		{"lte", spec.OpLE, true},
		{"ge", spec.OpGE, true},
		{"gte", spec.OpGE, true},
		{"lt", spec.OpLT, true},
		{"gt", spec.OpGT, true},
		{"eq", spec.OpEQ, true},
		{"≤", spec.OpLE, true},
		{"≥", spec.OpGE, true},
		{"!=", "", false},
		{"", "", false},
		{"invalid", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got, ok := spec.NormalizeOp(tc.input)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

// --- Op.UnmarshalText ---

func TestOp_UnmarshalText_Valid(t *testing.T) {
	var op spec.Op
	require.NoError(t, op.UnmarshalText([]byte("<=")))
	assert.Equal(t, spec.OpLE, op)
}

func TestOp_UnmarshalText_Invalid(t *testing.T) {
	var op spec.Op
	err := op.UnmarshalText([]byte("!="))
	assert.ErrorContains(t, err, "invalid op")
}

// --- PromMetric ---

func TestPromMetric_WithLabels(t *testing.T) {
	ref := spec.PromMetric("controller_runtime_reconcile_total", spec.Labels{"result": "error"})
	assert.Contains(t, ref.Key, "controller_runtime_reconcile_total")
	assert.Contains(t, ref.Key, `result="error"`)
}

func TestPromMetric_NoLabels(t *testing.T) {
	ref := spec.PromMetric("my_metric", spec.Labels{})
	assert.Equal(t, "my_metric", ref.Key)
}

// --- UnsafePromKey ---

func TestUnsafePromKey(t *testing.T) {
	ref := spec.UnsafePromKey(`my_metric{a="b"}`)
	assert.Equal(t, `my_metric{a="b"}`, ref.Key)
}
