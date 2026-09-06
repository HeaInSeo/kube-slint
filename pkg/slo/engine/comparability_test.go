package engine

import (
	"context"
	"testing"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/slo/fetch"
	"github.com/HeaInSeo/kube-slint/pkg/slo/spec"
	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
)

func e1WindowSpec(id, mode string) spec.SLISpec {
	return spec.SLISpec{
		ID: id, Unit: "ms", Kind: "latency",
		Inputs:  []spec.MetricRef{{Key: "request_ms"}},
		Compute: spec.ComputeSpec{Mode: spec.ComputeMode(mode)},
	}
}

func e1Run(t *testing.T, cfg RunConfig, specs ...spec.SLISpec) (*summary.Summary, error) {
	t.Helper()
	cfg.StartedAt = time.Unix(1000, 0)
	cfg.FinishedAt = time.Unix(1060, 0)
	eng := New(nil, &mockWriter{}, nil)
	return eng.Execute(context.Background(), ExecuteRequest{
		Config: cfg,
		Specs:  specs,
		WindowFetcher: &mockWindowFetcher{samples: []fetch.Sample{
			{At: time.Unix(1001, 0), Values: map[string]float64{"request_ms": 10}},
			{At: time.Unix(1002, 0), Values: map[string]float64{"request_ms": 20}},
		}},
		Reliability: &summary.Reliability{},
	})
}

func e1TrustContract() *TrustContract {
	return &TrustContract{SubjectID: "release-abc@sha256:deadbeef", SourceConfigID: "prom-default-v1", WindowID: "60m"}
}

// E1: the protected producer path emits exactly slo.v4 with a complete nonblank
// per-SLI comparability identity, and the artifact passes schema validation +
// carries the trust-correct contract KSL-T consumers require.
func TestExecute_ProtectedEmitsV4WithCompleteComparability(t *testing.T) {
	sum, err := e1Run(t, RunConfig{TrustContract: e1TrustContract()},
		e1WindowSpec("latency_avg", "window_avg"), e1WindowSpec("latency_p95", "window_p95"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if sum.SchemaVersion != summary.SchemaVersionTrust {
		t.Fatalf("schema = %q, want %q", sum.SchemaVersion, summary.SchemaVersionTrust)
	}
	if !summary.IsTrustCorrectContract(*sum) {
		t.Fatal("summary must report the trust-correct contract")
	}
	if err := summary.Validate(*sum); err != nil {
		t.Fatalf("produced v4 summary must pass schema validation: %v", err)
	}
	for _, r := range sum.Results {
		if !r.Comparability.Complete() {
			t.Fatalf("result %q must carry a complete comparability identity, got %+v", r.ID, r.Comparability)
		}
		if r.Comparability.SubjectID != "release-abc@sha256:deadbeef" ||
			r.Comparability.SourceConfigID != "prom-default-v1" {
			t.Fatalf("result %q caller coordinates not propagated: %+v", r.ID, r.Comparability)
		}
	}
	// Distinct SLIs (different measurement semantics) must have distinct contract ids.
	if sum.Results[0].Comparability.SLIContractID == sum.Results[1].Comparability.SLIContractID {
		t.Fatal("distinct SLIs must have distinct SLIContractID")
	}
}

// E1: without a TrustContract the producer stays on the legacy slo.v3 contract and
// stamps no comparability — historical output is never silently upgraded.
func TestExecute_LegacyStaysV3NoComparability(t *testing.T) {
	sum, err := e1Run(t, RunConfig{}, e1WindowSpec("latency_avg", "window_avg"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if sum.SchemaVersion != summary.SchemaVersion {
		t.Fatalf("schema = %q, want legacy %q", sum.SchemaVersion, summary.SchemaVersion)
	}
	for _, r := range sum.Results {
		if r.Comparability != nil {
			t.Fatalf("legacy result %q must not carry comparability", r.ID)
		}
	}
}

// E1: a protected run missing a caller-authoritative coordinate fails closed — it
// cannot silently produce a trust-correct artifact.
func TestExecute_ProtectedMissingCoordinateFailsClosed(t *testing.T) {
	for _, tc := range []*TrustContract{
		{SubjectID: "", SourceConfigID: "src", WindowID: "60m"},
		{SubjectID: "subj", SourceConfigID: "", WindowID: "60m"},
		{SubjectID: "subj", SourceConfigID: "src", WindowID: ""},
		{SubjectID: "subj", SourceConfigID: "src", WindowID: "   "},
	} {
		if _, err := e1Run(t, RunConfig{TrustContract: tc}, e1WindowSpec("x", "window_avg")); err == nil {
			t.Fatalf("protected run with incomplete coordinates %+v must fail closed", tc)
		}
	}
}

// E1: a protected run whose collection FAILS is emitted as legacy slo.v3 (a failed
// measurement is not protected evidence), never as a v4 artifact that carries no
// comparability identity for the requested SLIs.
func TestExecute_ProtectedCollectionFailureStaysV3(t *testing.T) {
	cfg := RunConfig{TrustContract: e1TrustContract()}
	cfg.StartedAt = time.Unix(1000, 0)
	cfg.FinishedAt = time.Unix(1060, 0)
	eng := New(nil, &mockWriter{}, nil) // no point MetricsFetcher -> point collection fails
	pointSpec := spec.SLISpec{
		ID: "err_delta", Unit: "count", Kind: "errors",
		Inputs:  []spec.MetricRef{{Key: "errors_total"}},
		Compute: spec.ComputeSpec{Mode: spec.ComputeDelta},
	}
	sum, err := eng.Execute(context.Background(), ExecuteRequest{
		Config: cfg, Specs: []spec.SLISpec{pointSpec}, Reliability: &summary.Reliability{},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if sum.SchemaVersion != summary.SchemaVersion {
		t.Fatalf("failed protected collection must stay legacy %q, got %q", summary.SchemaVersion, sum.SchemaVersion)
	}
	for _, r := range sum.Results {
		if r.Comparability != nil {
			t.Fatalf("a failed-collection result must not carry a v4 comparability identity: %+v", r.Comparability)
		}
	}
}

// E1: measurement-contract identity derives from measurement semantics, not Gate
// Policy / advisory judgment, and not run-scoped data.
func TestSLIContractID_MeasurementSemanticsOnly(t *testing.T) {
	base := e1WindowSpec("lat", "window_avg")
	id := sliContractID(base)
	if id != sliContractID(base) {
		t.Fatal("sliContractID must be deterministic")
	}
	// Semantic changes → identity changes.
	changed := map[string]spec.SLISpec{
		"id":      func() spec.SLISpec { s := base; s.ID = "lat2"; return s }(),
		"unit":    func() spec.SLISpec { s := base; s.Unit = "s"; return s }(),
		"kind":    func() spec.SLISpec { s := base; s.Kind = "gauge"; return s }(),
		"compute": func() spec.SLISpec { s := base; s.Compute = spec.ComputeSpec{Mode: spec.ComputeWindowP95}; return s }(),
		"inputs":  func() spec.SLISpec { s := base; s.Inputs = []spec.MetricRef{{Key: "other_ms"}}; return s }(),
	}
	for name, s := range changed {
		if sliContractID(s) == id {
			t.Fatalf("a %s change must change SLIContractID", name)
		}
	}
	// Gate/advisory (Judge) change must NOT change measurement-contract identity.
	withJudge := base
	withJudge.Judge = &spec.JudgeSpec{}
	if sliContractID(withJudge) != id {
		t.Fatal("an advisory Judge change must NOT change SLIContractID (KSL-E1 property 3)")
	}
	// Input Alias is cosmetic and must not change identity.
	aliased := base
	aliased.Inputs = []spec.MetricRef{{Key: "request_ms", Alias: "req"}}
	if sliContractID(aliased) != id {
		t.Fatal("a cosmetic input alias must not change SLIContractID")
	}
}

// E1 (P2 canonicalization): counter-reset policy affects SLIContractID only by its
// effect on the measured value, only under delta. Warn/Fail/empty are
// measurement-equivalent (they preserve the value and differ only in a
// non-authoritative verdict); NoGrade/Skip clear the value and so differ; under a
// non-delta mode the policy is inert and never changes identity.
func TestSLIContractID_CounterResetCanonicalization(t *testing.T) {
	delta := func(p spec.CounterResetPolicy) spec.SLISpec {
		return spec.SLISpec{
			ID: "d", Unit: "count", Kind: "errors",
			Inputs:  []spec.MetricRef{{Key: "errs"}},
			Compute: spec.ComputeSpec{Mode: spec.ComputeDelta, OnCounterReset: p},
		}
	}
	warn := sliContractID(delta(spec.CounterResetWarn))
	// Empty default == Warn, and Fail differs only in the advisory verdict.
	if sliContractID(delta("")) != warn {
		t.Fatal("empty counter-reset policy (default Warn) must not change SLIContractID")
	}
	if sliContractID(delta(spec.CounterResetFail)) != warn {
		t.Fatal("Fail vs Warn differ only in the non-authoritative verdict; SLIContractID must not change")
	}
	// NoGrade/Skip clear the value -> measurement-different.
	if sliContractID(delta(spec.CounterResetNoGrade)) == warn {
		t.Fatal("NoGrade clears the value and must change SLIContractID")
	}
	if sliContractID(delta(spec.CounterResetSkip)) == sliContractID(delta(spec.CounterResetWarn)) {
		t.Fatal("Skip clears the value and must change SLIContractID")
	}
	// Under a non-delta mode the counter-reset policy is inert.
	win := func(p spec.CounterResetPolicy) spec.SLISpec {
		s := e1WindowSpec("w", "window_avg")
		s.Compute.OnCounterReset = p
		return s
	}
	if sliContractID(win(spec.CounterResetFail)) != sliContractID(win(spec.CounterResetNoGrade)) {
		t.Fatal("counter-reset policy is inert for non-delta modes and must not change SLIContractID")
	}
}

// E1: WindowID is the window/aggregation semantics + the caller's explicit window
// extent, not elapsed runtime. An aggregation change or an extent change must change
// it; the same semantics + same extent are deterministic.
func TestWindowID_SemanticNotRuntime(t *testing.T) {
	avg := windowID(e1WindowSpec("lat", "window_avg"), "60m")
	p95 := windowID(e1WindowSpec("lat", "window_p95"), "60m")
	if avg == p95 {
		t.Fatal("a window/aggregation semantic change must change WindowID")
	}
	if avg != windowID(e1WindowSpec("lat", "window_avg"), "60m") {
		t.Fatal("WindowID must be deterministic for the same window semantics and extent")
	}
	// The caller's logical window extent is part of identity: a 5m and a 60m window
	// over the same aggregation must NOT compare as the same window.
	if windowID(e1WindowSpec("lat", "window_avg"), "5m") == avg {
		t.Fatal("a different logical window extent must change WindowID")
	}
}

// E1: subject and source-config coordinates are caller-driven; a change in either
// changes only its own coordinate. Run ID/timestamp alone never changes identity.
func TestComparability_CoordinateIndependenceAndRunScope(t *testing.T) {
	s := e1WindowSpec("lat", "window_avg")
	runA, err := e1Run(t, RunConfig{RunID: "runA", TrustContract: e1TrustContract()}, s)
	if err != nil {
		t.Fatalf("runA: %v", err)
	}
	// Same spec + same contract, different RunID → identical comparability (run scope
	// alone is not semantic identity).
	runB, err := e1Run(t, RunConfig{RunID: "runB", TrustContract: e1TrustContract()}, s)
	if err != nil {
		t.Fatalf("runB: %v", err)
	}
	if *runA.Results[0].Comparability != *runB.Results[0].Comparability {
		t.Fatal("run ID alone must not change comparability identity")
	}
	// Subject change → only SubjectID changes.
	subj, err := e1Run(t, RunConfig{TrustContract: &TrustContract{SubjectID: "other", SourceConfigID: "prom-default-v1", WindowID: "60m"}}, s)
	if err != nil {
		t.Fatalf("subj: %v", err)
	}
	base := runA.Results[0].Comparability
	got := subj.Results[0].Comparability
	if got.SubjectID == base.SubjectID {
		t.Fatal("a subject change must change SubjectID")
	}
	if got.SLIContractID != base.SLIContractID || got.WindowID != base.WindowID || got.SourceConfigID != base.SourceConfigID {
		t.Fatal("a subject change must not change the other coordinates")
	}
	// Source-config change → only SourceConfigID changes.
	src, err := e1Run(t, RunConfig{TrustContract: &TrustContract{SubjectID: "release-abc@sha256:deadbeef", SourceConfigID: "other-src", WindowID: "60m"}}, s)
	if err != nil {
		t.Fatalf("src: %v", err)
	}
	got = src.Results[0].Comparability
	if got.SourceConfigID == base.SourceConfigID {
		t.Fatal("a source-config change must change SourceConfigID")
	}
	if got.SLIContractID != base.SLIContractID || got.WindowID != base.WindowID || got.SubjectID != base.SubjectID {
		t.Fatal("a source-config change must not change the other coordinates")
	}
	// Window-extent change → only WindowID changes.
	win, err := e1Run(t, RunConfig{TrustContract: &TrustContract{SubjectID: "release-abc@sha256:deadbeef", SourceConfigID: "prom-default-v1", WindowID: "5m"}}, s)
	if err != nil {
		t.Fatalf("win: %v", err)
	}
	got = win.Results[0].Comparability
	if got.WindowID == base.WindowID {
		t.Fatal("a window-extent change must change WindowID")
	}
	if got.SLIContractID != base.SLIContractID || got.SubjectID != base.SubjectID || got.SourceConfigID != base.SourceConfigID {
		t.Fatal("a window-extent change must not change the other coordinates")
	}
}

// E1: applyTrustContract fails closed when a result has no spec to derive identity.
func TestApplyTrustContract_UnknownResultFailsClosed(t *testing.T) {
	sum := &summary.Summary{
		SchemaVersion: summary.SchemaVersionTrust,
		Results:       []summary.SLIResult{{ID: "orphan"}},
	}
	if err := applyTrustContract(sum, nil, e1TrustContract()); err == nil {
		t.Fatal("a result without a matching spec must fail closed")
	}
}
