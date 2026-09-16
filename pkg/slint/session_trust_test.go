package slint

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/slo/engine"
	"github.com/HeaInSeo/kube-slint/pkg/slo/spec"
	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// KSL-E1P — Protected Producer Uplift.
//
// These tests cover the bounded producer uplift that lets the public
// pkg/slint.Session producer emit the trust-correct slo.v4 contract by threading
// a caller-supplied engine.TrustContract through SessionConfig -> Session.End() ->
// engine.RunConfig.TrustContract. They assert the session-level behavior only; the
// engine's contract semantics (validation, comparability derivation) are owned and
// tested in pkg/slo/engine and are exercised here through the public Session path.

// trustSpecs returns a single window-avg SLI reading "request_ms", the same shape
// proven healthy by TestSession_End_PassesWindowFetcher, so collection never fails
// and a TrustContract run is emitted as slo.v4 rather than downgraded to v3.
func trustSpecs() []spec.SLISpec {
	return []spec.SLISpec{{
		ID:      "request_ms_avg",
		Inputs:  []spec.MetricRef{{Key: "request_ms"}},
		Compute: spec.ComputeSpec{Mode: spec.ComputeWindowAvg},
	}}
}

func newTrustSession(t *testing.T, tc *engine.TrustContract) *Session {
	t.Helper()
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := NewSession(SessionConfig{
		Namespace:     "measured-ns",
		TestCase:      "trust",
		Now:           func() time.Time { return now },
		Specs:         trustSpecs(),
		WindowFetcher: &mockWindowFetcher{},
		TrustContract: tc,
	})
	sess.Start()
	return sess
}

// A. Legacy preservation: a session with no TrustContract keeps emitting the
// legacy slo.v3 contract and stamps no per-SLI comparability identity.
func TestSession_End_NoTrustContract_EmitsLegacyV3(t *testing.T) {
	sess := newTrustSession(t, nil)

	sum, err := sess.End(context.Background())
	require.NoError(t, err)
	require.NotNil(t, sum)

	assert.Equal(t, summary.SchemaVersionLegacy, sum.SchemaVersion)
	require.Len(t, sum.Results, 1)
	assert.Nil(t, sum.Results[0].Comparability,
		"legacy slo.v3 output must carry no comparability identity")
}

// B. Protected path: a complete TrustContract makes the producer emit slo.v4 and
// stamp a complete per-SLI comparability identity on every result.
func TestSession_End_CompleteTrustContract_EmitsV4WithComparability(t *testing.T) {
	tc := &engine.TrustContract{
		SubjectID:      "sha256:release-abc",
		SourceConfigID: "srccfg-prod-01",
		WindowID:       "60m",
	}
	sess := newTrustSession(t, tc)

	sum, err := sess.End(context.Background())
	require.NoError(t, err)
	require.NotNil(t, sum)

	assert.Equal(t, summary.SchemaVersionTrust, sum.SchemaVersion)
	require.Len(t, sum.Results, 1)
	cmp := sum.Results[0].Comparability
	require.NotNil(t, cmp, "protected slo.v4 output must carry a comparability identity")
	assert.True(t, cmp.Complete(), "comparability identity must be complete")
	assert.NotEmpty(t, cmp.SLIContractID, "SLIContractID is derived from SLI semantics")
	assert.NotEmpty(t, cmp.WindowID, "WindowID is derived from the SLI + caller window")
}

// C. Incomplete-contract fail-closed: a protected request whose caller coordinates
// are incomplete (any one of SubjectID/SourceConfigID/WindowID blank) must error
// and must not produce a slo.v4 artifact. Each missing coordinate is checked.
func TestSession_End_IncompleteTrustContract_FailsClosed(t *testing.T) {
	cases := map[string]*engine.TrustContract{
		"missing SubjectID":      {SubjectID: "", SourceConfigID: "srccfg-01", WindowID: "60m"},
		"missing SourceConfigID": {SubjectID: "subj-01", SourceConfigID: "", WindowID: "60m"},
		"missing WindowID":       {SubjectID: "subj-01", SourceConfigID: "srccfg-01", WindowID: ""},
		"blank whitespace only":  {SubjectID: "  ", SourceConfigID: "srccfg-01", WindowID: "60m"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			sess := newTrustSession(t, tc)

			sum, err := sess.End(context.Background())
			require.Error(t, err, "incomplete protected contract must fail closed")
			if sum != nil {
				assert.NotEqual(t, summary.SchemaVersionTrust, sum.SchemaVersion,
					"a failed-closed protected run must not yield a slo.v4 artifact")
			}
		})
	}
}

// C2. Fail-closed leaves nothing on disk: an incomplete protected contract must
// not write ANY slo.v4 artifact, even when the session is configured to write
// artifacts (ArtifactsDir set). This pins the on-disk half of I3 — the engine
// validates before its sole write, and the session's static-alias write only runs
// after a successful ExecuteStandard.
func TestSession_End_IncompleteTrustContract_WritesNoArtifact(t *testing.T) {
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	dir := t.TempDir()
	sess := NewSession(SessionConfig{
		Namespace:     "measured-ns",
		TestCase:      "trust-nodisk",
		ArtifactsDir:  dir,
		Now:           func() time.Time { return now },
		Specs:         trustSpecs(),
		WindowFetcher: &mockWindowFetcher{},
		TrustContract: &engine.TrustContract{SubjectID: "subj-01", SourceConfigID: "", WindowID: "60m"},
	})
	sess.Start()

	_, err := sess.End(context.Background())
	require.Error(t, err, "incomplete protected contract must fail closed")

	entries, readErr := os.ReadDir(dir)
	require.NoError(t, readErr)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		b, rErr := os.ReadFile(filepath.Join(dir, e.Name()))
		require.NoError(t, rErr)
		assert.NotContains(t, string(b), summary.SchemaVersionTrust,
			"fail-closed run must not leave a slo.v4 artifact on disk (%s)", e.Name())
	}
}

// Regression (Codex P1): a fail-closed protected run must invalidate a previous
// successful run's static alias when the ArtifactsDir is reused, so the stale
// slo.v4 sli-summary.json is never read as this run's current evidence. Without
// the fix, ExecuteStandard errors before the alias is rewritten and the old v4
// file survives at the default slint-gate input path.
func TestSession_End_ProtectedFailure_InvalidatesStaleStaticAlias(t *testing.T) {
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	dir := t.TempDir()
	alias := filepath.Join(dir, "sli-summary.json")

	// Run 1: a successful protected run leaves a real slo.v4 static alias.
	ok := NewSession(SessionConfig{
		Namespace:     "measured-ns",
		TestCase:      "ok",
		RunID:         "run-ok",
		ArtifactsDir:  dir,
		Now:           func() time.Time { return now },
		Specs:         trustSpecs(),
		WindowFetcher: &mockWindowFetcher{},
		TrustContract: &engine.TrustContract{SubjectID: "subj-01", SourceConfigID: "srccfg-01", WindowID: "60m"},
	})
	ok.Start()
	okSum, err := ok.End(context.Background())
	require.NoError(t, err)
	require.NotNil(t, okSum)
	require.Equal(t, summary.SchemaVersionTrust, okSum.SchemaVersion)

	priorBytes, readErr := os.ReadFile(alias)
	require.NoError(t, readErr, "successful protected run must have written the static alias")
	require.Contains(t, string(priorBytes), summary.SchemaVersionTrust,
		"precondition: the reused dir holds a prior successful slo.v4 alias")

	// Run 2: reuse the SAME dir with an incomplete protected contract. The engine
	// fails closed before the alias is rewritten; the prior v4 alias must not
	// survive as current evidence.
	bad := NewSession(SessionConfig{
		Namespace:     "measured-ns",
		TestCase:      "bad",
		RunID:         "run-bad",
		ArtifactsDir:  dir,
		Now:           func() time.Time { return now },
		Specs:         trustSpecs(),
		WindowFetcher: &mockWindowFetcher{},
		TrustContract: &engine.TrustContract{SubjectID: "subj-01", SourceConfigID: "", WindowID: "60m"},
	})
	bad.Start()
	_, err = bad.End(context.Background())
	require.Error(t, err, "incomplete protected contract must fail closed")

	_, statErr := os.Stat(alias)
	assert.True(t, os.IsNotExist(statErr),
		"a fail-closed protected run must invalidate the prior successful static alias; found it still present at %s", alias)
}

// aliasFailingWriter delegates every write to inner EXCEPT the static alias
// ("sli-summary.json"), which it fails — simulating a successful run whose current
// summary cannot be published to the default gate input path.
type aliasFailingWriter struct{ inner summary.Writer }

func (w aliasFailingWriter) Write(path string, s summary.Summary) error {
	if filepath.Base(path) == "sli-summary.json" {
		return errStaticAliasWrite
	}
	return w.inner.Write(path, s)
}

var errStaticAliasWrite = errWrite("simulated static alias write failure")

type errWrite string

func (e errWrite) Error() string { return string(e) }

// Regression (Codex P1, success-path twin): even when the run itself SUCCEEDS, if
// the current summary cannot be published to the static alias, End() must not leave
// a prior successful run's alias there as current evidence and must not report
// success. It invalidates the stale alias and returns an error.
func TestSession_End_StaticAliasWriteFailure_InvalidatesStaleAliasAndErrors(t *testing.T) {
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	dir := t.TempDir()
	alias := filepath.Join(dir, "sli-summary.json")

	// Run 1: a successful protected run leaves a real slo.v4 static alias.
	ok := NewSession(SessionConfig{
		Namespace:     "measured-ns",
		TestCase:      "ok",
		RunID:         "run-ok",
		ArtifactsDir:  dir,
		Now:           func() time.Time { return now },
		Specs:         trustSpecs(),
		WindowFetcher: &mockWindowFetcher{},
		TrustContract: &engine.TrustContract{SubjectID: "subj-01", SourceConfigID: "srccfg-01", WindowID: "60m"},
	})
	ok.Start()
	_, err := ok.End(context.Background())
	require.NoError(t, err)
	priorBytes, readErr := os.ReadFile(alias)
	require.NoError(t, readErr)
	require.Contains(t, string(priorBytes), summary.SchemaVersionTrust,
		"precondition: the reused dir holds a prior successful slo.v4 alias")

	// Run 2: reuse the dir; ExecuteStandard succeeds but the static-alias write
	// fails. The stale prior alias must be invalidated and End must error.
	bad := NewSession(SessionConfig{
		Namespace:     "measured-ns",
		TestCase:      "bad",
		RunID:         "run-bad",
		ArtifactsDir:  dir,
		Now:           func() time.Time { return now },
		Specs:         trustSpecs(),
		WindowFetcher: &mockWindowFetcher{},
		Writer:        aliasFailingWriter{inner: summary.NewJSONFileWriter()},
		TrustContract: &engine.TrustContract{SubjectID: "subj-01", SourceConfigID: "srccfg-01", WindowID: "60m"},
	})
	bad.Start()
	_, err = bad.End(context.Background())
	require.Error(t, err, "a failed static-alias write must surface as an End() error")

	_, statErr := os.Stat(alias)
	assert.True(t, os.IsNotExist(statErr),
		"a failed static-alias write must invalidate the prior successful alias; found it still present at %s", alias)
}

// Regression (Codex P1 4014126101): a run that fails to allocate its per-run
// unique path (NextSummaryPath) must still invalidate a prior successful run's
// static alias, so the stale slo.v4 sli-summary.json is not read as this run's
// current evidence. The failure is triggered through the production path: an
// accepted-but-long RunID+TestCase sanitizes to a ~258-byte unique filename that
// exceeds the 255-byte component limit, so os.Stat inside NextSummaryPath returns
// ENAMETOOLONG (a non-IsNotExist error) before ExecuteStandard runs.
func TestSession_End_NextSummaryPathFailure_InvalidatesStaleStaticAlias(t *testing.T) {
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	dir := t.TempDir()
	alias := filepath.Join(dir, "sli-summary.json")

	// Run 1: a successful protected run (short names) leaves a real slo.v4 alias.
	ok := NewSession(SessionConfig{
		Namespace:     "measured-ns",
		TestCase:      "ok",
		RunID:         "run-ok",
		ArtifactsDir:  dir,
		Now:           func() time.Time { return now },
		Specs:         trustSpecs(),
		WindowFetcher: &mockWindowFetcher{},
		TrustContract: &engine.TrustContract{SubjectID: "subj-01", SourceConfigID: "srccfg-01", WindowID: "60m"},
	})
	ok.Start()
	_, err := ok.End(context.Background())
	require.NoError(t, err)
	priorBytes, readErr := os.ReadFile(alias)
	require.NoError(t, readErr)
	require.Contains(t, string(priorBytes), summary.SchemaVersionTrust,
		"precondition: the reused dir holds a prior successful slo.v4 alias")

	// Run 2: reuse the dir with an accepted-but-long RunID + TestCase. Each
	// sanitizes to 120 bytes, so the unique filename
	// "sli-summary.<120>.<120>.json" is 258 bytes > NAME_MAX and NextSummaryPath's
	// os.Stat fails with ENAMETOOLONG before any summary is produced. The static
	// alias ("sli-summary.json", short) must be invalidated and End must error.
	longRunID := strings.Repeat("r", 130)
	longTestCase := strings.Repeat("t", 130)
	bad := NewSession(SessionConfig{
		Namespace:     "measured-ns",
		TestCase:      longTestCase,
		RunID:         longRunID,
		ArtifactsDir:  dir,
		Now:           func() time.Time { return now },
		Specs:         trustSpecs(),
		WindowFetcher: &mockWindowFetcher{},
		TrustContract: &engine.TrustContract{SubjectID: "subj-01", SourceConfigID: "srccfg-01", WindowID: "60m"},
	})
	bad.Start()
	_, err = bad.End(context.Background())
	require.Error(t, err, "a NextSummaryPath allocation failure must surface as an End() error")

	_, statErr := os.Stat(alias)
	assert.True(t, os.IsNotExist(statErr),
		"a NextSummaryPath-failure run must invalidate the prior successful static alias; found it still present at %s", alias)
}

// D. No producer reinterpretation: the caller's authoritative coordinates reach the
// engine verbatim. SubjectID/SourceConfigID are deliberately distinct from the
// session's Namespace/RunID/Tags so the test would fail if the producer inferred or
// overrode them from run context instead of forwarding the caller's values.
func TestSession_End_CallerCoordinatesForwardedVerbatim(t *testing.T) {
	tc := &engine.TrustContract{
		SubjectID:      "subject-not-the-namespace",
		SourceConfigID: "source-config-not-the-runid",
		WindowID:       "explicit-window-90m",
	}
	sess := newTrustSession(t, tc)

	sum, err := sess.End(context.Background())
	require.NoError(t, err)
	require.Len(t, sum.Results, 1)
	cmp := sum.Results[0].Comparability
	require.NotNil(t, cmp)

	assert.Equal(t, "subject-not-the-namespace", cmp.SubjectID,
		"SubjectID must be the caller's value, never derived from Namespace")
	assert.Equal(t, "source-config-not-the-runid", cmp.SourceConfigID,
		"SourceConfigID must be the caller's value, never derived from RunID")
	// WindowID is hashed with the SLI's measurement mode, so it is not the raw
	// string, but it must be non-empty and stable for the caller's explicit window.
	assert.NotEmpty(t, cmp.WindowID)
}

// E. Existing engine protections survive on the Session path: a protected request
// with duplicate SLI IDs is rejected before any slo.v4 artifact is produced, exactly
// as it is on the direct engine path (validateProtectedSpecIDs).
func TestSession_End_ProtectedDuplicateSLIID_FailsClosed(t *testing.T) {
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	dup := spec.SLISpec{
		ID:      "request_ms_avg",
		Inputs:  []spec.MetricRef{{Key: "request_ms"}},
		Compute: spec.ComputeSpec{Mode: spec.ComputeWindowAvg},
	}
	sess := NewSession(SessionConfig{
		Namespace:     "measured-ns",
		TestCase:      "trust-dup",
		Now:           func() time.Time { return now },
		Specs:         []spec.SLISpec{dup, dup},
		WindowFetcher: &mockWindowFetcher{},
		TrustContract: &engine.TrustContract{
			SubjectID:      "subj-01",
			SourceConfigID: "srccfg-01",
			WindowID:       "60m",
		},
	})
	sess.Start()

	sum, err := sess.End(context.Background())
	require.Error(t, err, "protected run with duplicate SLI IDs must fail closed")
	if sum != nil {
		assert.NotEqual(t, summary.SchemaVersionTrust, sum.SchemaVersion)
	}
}

// F. Session regression: threading the TrustContract does not disturb the measured
// value or the rest of the Session pipeline. A protected run and a legacy run over
// the same fetcher/specs must compute the identical SLI value; only the contract
// version and comparability presence differ.
func TestSession_End_TrustContractDoesNotChangeMeasuredValue(t *testing.T) {
	legacy := newTrustSession(t, nil)
	legacySum, err := legacy.End(context.Background())
	require.NoError(t, err)
	require.Len(t, legacySum.Results, 1)
	require.NotNil(t, legacySum.Results[0].Value)

	protected := newTrustSession(t, &engine.TrustContract{
		SubjectID:      "subj-01",
		SourceConfigID: "srccfg-01",
		WindowID:       "60m",
	})
	protectedSum, err := protected.End(context.Background())
	require.NoError(t, err)
	require.Len(t, protectedSum.Results, 1)
	require.NotNil(t, protectedSum.Results[0].Value)

	assert.Equal(t, *legacySum.Results[0].Value, *protectedSum.Results[0].Value,
		"threading a TrustContract must not change the measured value")
	assert.Equal(t, summary.SchemaVersionLegacy, legacySum.SchemaVersion)
	assert.Equal(t, summary.SchemaVersionTrust, protectedSum.SchemaVersion)
}
