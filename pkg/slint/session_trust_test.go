package slint

import (
	"context"
	"os"
	"path/filepath"
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
