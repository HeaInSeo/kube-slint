package slint_test

import (
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/slint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSession_Smoke(t *testing.T) {
	t.Setenv("SLINT_DISABLE_DISCOVERY", "1")

	sess := slint.NewSession(slint.SessionConfig{
		Namespace:          "test-ns",
		MetricsServiceName: "test-svc",
		RunID:              "pkg-slint-test",
	})
	require.NotNil(t, sess)
}

func TestDefaultSpecs_NotEmpty(t *testing.T) {
	specs := slint.DefaultSpecs()
	assert.NotEmpty(t, specs)
}

func TestBaselineSpecs_SameAsDefaultSpecs(t *testing.T) {
	assert.Equal(t, slint.DefaultSpecs(), slint.BaselineSpecs())
}
