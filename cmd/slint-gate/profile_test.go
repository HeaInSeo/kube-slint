package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validCustomProfile = `schema_version: "slint.profile.v1"
name: "my-custom-profile"
description: "a test profile"
candidates:
  - id: "some_metric_delta"
    operator: "=="
    value: 0
    tier: "core"
    reason: "test reason"
`

func TestLoadCustomProfile_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profile.yaml")
	require.NoError(t, os.WriteFile(path, []byte(validCustomProfile), 0o644))

	candidates, name, err := loadCustomProfile(path)
	require.NoError(t, err)
	assert.Equal(t, "my-custom-profile", name)
	require.Len(t, candidates, 1)
	assert.Equal(t, "some_metric_delta", candidates[0].ID)
	assert.Equal(t, tierCore, candidates[0].Tier)
}

func TestLoadCustomProfile_DefaultTierIsCore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profile.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`schema_version: "slint.profile.v1"
name: "p"
candidates:
  - id: "m"
    operator: "=="
    value: 0
`), 0o644))

	candidates, _, err := loadCustomProfile(path)
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	assert.Equal(t, tierCore, candidates[0].Tier)
}

func TestLoadCustomProfile_UnsupportedSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profile.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`schema_version: "slint.profile.v99"
candidates:
  - id: "m"
    operator: "=="
`), 0o644))

	_, _, err := loadCustomProfile(path)
	assert.Error(t, err)
}

func TestLoadCustomProfile_UnknownOperator(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profile.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`schema_version: "slint.profile.v1"
candidates:
  - id: "m"
    operator: "!="
    value: 0
`), 0o644))

	_, _, err := loadCustomProfile(path)
	assert.Error(t, err)
}

func TestLoadCustomProfile_UnrecognizedTier(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profile.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`schema_version: "slint.profile.v1"
candidates:
  - id: "m"
    operator: "=="
    value: 0
    tier: "bogus"
`), 0o644))

	_, _, err := loadCustomProfile(path)
	assert.Error(t, err)
}

func TestLoadCustomProfile_InformationalTierSkipsOperatorCheck(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profile.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`schema_version: "slint.profile.v1"
candidates:
  - id: "m"
    tier: "informational"
`), 0o644))

	candidates, _, err := loadCustomProfile(path)
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	assert.Equal(t, tierInformational, candidates[0].Tier)
}

func TestLoadCustomProfile_EmptyID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profile.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`schema_version: "slint.profile.v1"
candidates:
  - id: ""
    operator: "=="
    value: 0
`), 0o644))

	_, _, err := loadCustomProfile(path)
	assert.Error(t, err)
}

func TestResolveProfileCandidates_ExplicitFileWinsOverEverything(t *testing.T) {
	dir := t.TempDir()
	explicit := filepath.Join(dir, "explicit.yaml")
	require.NoError(t, os.WriteFile(explicit, []byte(validCustomProfile), 0o644))

	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".slint", "profiles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".slint", "profiles", "kubebuilder-operator.yaml"), []byte(`schema_version: "slint.profile.v1"
name: "local-convention"
candidates:
  - id: "x"
    operator: "=="
    value: 0
`), 0o644))

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(oldWd) }()

	candidates, name, err := resolveProfileCandidates(explicit, "kubebuilder-operator")
	require.NoError(t, err)
	assert.Equal(t, "my-custom-profile", name)
	require.Len(t, candidates, 1)
	assert.Equal(t, "some_metric_delta", candidates[0].ID)
}

func TestResolveProfileCandidates_LocalConventionWinsOverBuiltin(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".slint", "profiles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".slint", "profiles", "kubebuilder-operator.yaml"), []byte(`schema_version: "slint.profile.v1"
name: "local-convention"
candidates:
  - id: "x"
    operator: "=="
    value: 0
`), 0o644))

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(oldWd) }()

	candidates, name, err := resolveProfileCandidates("", "kubebuilder-operator")
	require.NoError(t, err)
	assert.Equal(t, "local-convention", name)
	require.Len(t, candidates, 1)
	assert.Equal(t, "x", candidates[0].ID)
}

func TestResolveProfileCandidates_FallsBackToBuiltin(t *testing.T) {
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(oldWd) }()

	candidates, name, err := resolveProfileCandidates("", "kubebuilder-operator")
	require.NoError(t, err)
	assert.Equal(t, "kubebuilder-operator", name)
	assert.Len(t, candidates, len(kubebuilderOperatorCandidates))
}

func TestResolveProfileCandidates_UnknownBuiltinProfile(t *testing.T) {
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(oldWd) }()

	_, _, err = resolveProfileCandidates("", "bogus")
	assert.Error(t, err)
}
