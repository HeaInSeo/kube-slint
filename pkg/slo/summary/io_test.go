package summary_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validSummary() summary.Summary {
	v := 3.0
	return summary.Summary{
		SchemaVersion: summary.SchemaVersion,
		GeneratedAt:   time.Now(),
		Results: []summary.SLIResult{
			{ID: "reconcile_total_delta", Value: &v, Status: summary.StatusPass},
		},
	}
}

func writeTempJSON(t *testing.T, dir string, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	path := filepath.Join(dir, "summary.json")
	require.NoError(t, os.WriteFile(path, data, 0o644))
	return path
}

// --- LoadFile ---

func TestLoadFile_Valid(t *testing.T) {
	dir := t.TempDir()
	path := writeTempJSON(t, dir, validSummary())

	s, err := summary.LoadFile(path)
	require.NoError(t, err)
	assert.Equal(t, summary.SchemaVersion, s.SchemaVersion)
	require.Len(t, s.Results, 1)
	assert.Equal(t, "reconcile_total_delta", s.Results[0].ID)
}

func TestLoadFile_Missing(t *testing.T) {
	_, err := summary.LoadFile("/nonexistent/path/summary.json")
	assert.Error(t, err)
}

func TestLoadFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(path, []byte("{not json{{"), 0o644))

	_, err := summary.LoadFile(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON")
}

func TestLoadFile_EmptySchemaVersion(t *testing.T) {
	dir := t.TempDir()
	s := validSummary()
	s.SchemaVersion = ""
	path := writeTempJSON(t, dir, s)

	_, err := summary.LoadFile(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schemaVersion is empty")
}

func TestLoadFile_UnsupportedSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	s := validSummary()
	s.SchemaVersion = "slint.summary.v99"
	path := writeTempJSON(t, dir, s)

	_, err := summary.LoadFile(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported schemaVersion")
}

// --- WriteFile ---

func TestWriteFile_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")
	orig := validSummary()

	require.NoError(t, summary.WriteFile(path, orig))

	loaded, err := summary.LoadFile(path)
	require.NoError(t, err)
	assert.Equal(t, orig.SchemaVersion, loaded.SchemaVersion)
	assert.Equal(t, orig.Results[0].ID, loaded.Results[0].ID)
}

func TestWriteFile_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "dir", "out.json")

	require.NoError(t, summary.WriteFile(path, validSummary()))
	assert.FileExists(t, path)
}

// --- Validate ---

func TestValidate_Valid(t *testing.T) {
	assert.NoError(t, summary.Validate(validSummary()))
}

func TestValidate_EmptySchemaVersion(t *testing.T) {
	s := validSummary()
	s.SchemaVersion = ""
	assert.Error(t, summary.Validate(s))
}

func TestValidate_ZeroGeneratedAt(t *testing.T) {
	s := validSummary()
	s.GeneratedAt = time.Time{}
	err := summary.Validate(s)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "generatedAt is zero")
}

func TestValidate_EmptyResultID(t *testing.T) {
	s := validSummary()
	s.Results[0].ID = ""
	err := summary.Validate(s)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "results[0].id is empty")
}

func TestValidate_DuplicateResultID(t *testing.T) {
	s := validSummary()
	dup := s.Results[0]
	s.Results = append(s.Results, dup)
	err := summary.Validate(s)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate result ID")
}

func TestValidate_UnknownResultStatus(t *testing.T) {
	s := validSummary()
	s.Results[0].Status = "bogus"
	err := summary.Validate(s)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a recognized status")
}

func TestValidate_AllKnownStatusesAccepted(t *testing.T) {
	for _, st := range []summary.Status{
		summary.StatusPass, summary.StatusWarn, summary.StatusFail, summary.StatusBlock, summary.StatusSkip,
	} {
		s := validSummary()
		s.Results[0].Status = st
		assert.NoError(t, summary.Validate(s), "status %q should be valid", st)
	}
}
