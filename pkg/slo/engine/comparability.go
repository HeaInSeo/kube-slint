package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"sort"
	"strings"

	"github.com/HeaInSeo/kube-slint/pkg/slo/spec"
	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
)

// contractVersion returns the measurement contract version to stamp for a run: the
// trust-correct slo.v4 only when a TrustContract is present AND collection did not
// fail, otherwise legacy slo.v3. A failed collection is not protected evidence, so
// it is emitted as legacy v3 rather than as a v4 artifact that carries no
// comparability identity for the requested SLIs (KSL-E1). Historical/unprotected
// output is never silently reinterpreted as v4.
func contractVersion(cfg RunConfig, rel *summary.Reliability) string {
	if cfg.TrustContract == nil {
		return summary.SchemaVersion
	}
	if rel != nil && strings.EqualFold(strings.TrimSpace(rel.CollectionStatus), "Failed") {
		return summary.SchemaVersion
	}
	return summary.SchemaVersionTrust
}

// validateTrustContract fails closed on a protected request whose caller-supplied
// authoritative coordinates are incomplete, so a trust-correct artifact can never
// be produced silently without a complete identity (KSL-E1).
func validateTrustContract(tc *TrustContract) error {
	if tc == nil {
		return nil
	}
	if strings.TrimSpace(tc.SubjectID) == "" {
		return fmt.Errorf("trust-correct measurement requires a non-empty SubjectID (the evidence subject is caller-authoritative)")
	}
	if strings.TrimSpace(tc.SourceConfigID) == "" {
		return fmt.Errorf("trust-correct measurement requires a non-empty SourceConfigID")
	}
	if strings.TrimSpace(tc.WindowID) == "" {
		return fmt.Errorf("trust-correct measurement requires a non-empty WindowID (the logical measurement window is caller-authoritative and must not be derived from run timestamps)")
	}
	return nil
}

// applyTrustContract stamps a complete per-SLI comparability identity on every
// result of a trust-correct (slo.v4) summary, failing closed if any identity is
// incomplete or a result has no spec to derive from. It is a no-op when no
// TrustContract is set (legacy slo.v3).
//
// SLIContractID and WindowID are derived from each SLI's MEASUREMENT semantics;
// SubjectID and SourceConfigID come from the caller's TrustContract. Gate policy /
// threshold / producer-verdict inputs are deliberately excluded from identity
// (KSL-E1 property 3).
func applyTrustContract(sum *summary.Summary, specs []spec.SLISpec, tc *TrustContract) error {
	if tc == nil {
		return nil
	}
	if err := validateTrustContract(tc); err != nil {
		return err
	}
	specByID := make(map[string]spec.SLISpec, len(specs))
	for _, s := range specs {
		specByID[s.ID] = s
	}
	for i := range sum.Results {
		s, ok := specByID[sum.Results[i].ID]
		if !ok {
			return fmt.Errorf(
				"trust-correct measurement: no spec for result %q; cannot derive comparability identity", sum.Results[i].ID)
		}
		cmp := &summary.Comparability{
			SLIContractID:  sliContractID(s),
			SubjectID:      tc.SubjectID,
			WindowID:       windowID(s, tc.WindowID),
			SourceConfigID: tc.SourceConfigID,
		}
		if !cmp.Complete() {
			return fmt.Errorf("trust-correct measurement: incomplete comparability identity for %q", sum.Results[i].ID)
		}
		sum.Results[i].Comparability = cmp
	}
	return nil
}

// sliContractID derives a deterministic identity for an SLI's MEASUREMENT contract
// semantics — what is measured and how: SLI id, unit, kind, its input source keys,
// its compute mode/aggregation, and (for delta only) the counter-reset policy's
// EFFECT ON THE MEASURED VALUE. It deliberately excludes Judge/threshold/policy
// (which judge the value, not how it is measured), the input Alias (a cosmetic
// display name), the advisory-only difference between counter-reset policies that
// keep the same value, and any run-scoped data.
//
// Input order is part of the measurement identity for every mode that combines its
// inputs into the value: evalSLI and the window min/max/avg/percentile loops
// accumulate float64 in the supplied order, and floating-point addition is not
// associative (e.g. 1e16, -1e16, 1 sums to 1 or 0 depending on order), so reordering
// can change the measured value — sorting would let specs that measure DIFFERENTLY
// share an identity (a false regression/pass). The one exception is window_ratio,
// whose value uses only inputs[0] (numerator) and inputs[1] (denominator); any
// inputs[2:] are an unordered required-present set (the windowValues evidence check),
// not part of the value, so their order is canonicalized while positions 0/1 are
// kept, avoiding a false BASELINE_INCOMPARABLE from a mere tail reorder.
func sliContractID(s spec.SLISpec) string {
	h := sha256.New()
	writeField(h, "kube-slint.sli.contract.v1")
	writeField(h, s.ID)
	writeField(h, s.Unit)
	writeField(h, s.Kind)
	mode := canonicalMeasurementMode(s.Compute.Mode)
	writeField(h, string(mode))
	// Counter-reset policy only affects the measured value under delta, and there
	// only via whether the value is preserved or cleared — the warn/fail choice is
	// a non-authoritative producer verdict, so it must not change identity.
	if mode == spec.ComputeDelta {
		writeField(h, "reset="+counterResetMeasurementEffect(s.Compute.OnCounterReset))
	}
	writeField(h, fmt.Sprintf("inputs=%d", len(s.Inputs)))
	if mode == spec.ComputeWindowRatio && len(s.Inputs) > 2 {
		keys := make([]string, len(s.Inputs))
		for i, in := range s.Inputs {
			keys[i] = in.Key
		}
		sort.Strings(keys[2:]) // tail is an unordered required set, not part of the value
		for _, k := range keys {
			writeField(h, k)
		}
	} else {
		for _, in := range s.Inputs {
			writeField(h, in.Key)
		}
	}
	return "slic-v1-" + hex.EncodeToString(h.Sum(nil))[:32]
}

// canonicalMeasurementMode folds compute modes that produce an identical
// measurement onto one representative, so a non-semantic spelling change does not
// change identity. The legacy ComputeSingle ("single") is evaluated identically to
// ComputeStart (evalSLI handles them in one fallthrough branch: value = start
// snapshot), so migrating an otherwise-unchanged SLI from "single" to "start" must
// keep the same SLIContractID/WindowID rather than reporting BASELINE_INCOMPARABLE.
func canonicalMeasurementMode(m spec.ComputeMode) spec.ComputeMode {
	if m == spec.ComputeSingle {
		return spec.ComputeStart
	}
	return m
}

// counterResetMeasurementEffect canonicalizes a counter-reset policy to its effect
// on the MEASURED value, dropping the advisory producer verdict: Warn (the empty
// default) and Fail both preserve the value and differ only in a non-authoritative
// status, so they are measurement-equivalent; NoGrade and Skip both clear the value
// (measurement unreliable).
func counterResetMeasurementEffect(p spec.CounterResetPolicy) string {
	switch p {
	case spec.CounterResetNoGrade, spec.CounterResetSkip:
		return "clear"
	default: // CounterResetWarn, CounterResetFail, "" (default Warn)
		return "preserve"
	}
}

// windowID derives a deterministic identity for the window/aggregation semantics of
// an SLI: its compute mode (point-vs-window and the aggregation, e.g. window_p95)
// together with the caller's explicit logical window extent. The window extent is
// caller-authoritative (KSL-E1 property 5) and is NEVER derived from
// StartedAt/FinishedAt elapsed runtime, so two runs over different window extents
// (e.g. 5m vs 60m) receive different WindowIDs and are never compared as if
// measured over the same window.
func windowID(s spec.SLISpec, logicalWindow string) string {
	h := sha256.New()
	writeField(h, "kube-slint.sli.window.v1")
	writeField(h, string(canonicalMeasurementMode(s.Compute.Mode)))
	writeField(h, logicalWindow)
	return "win-v1-" + hex.EncodeToString(h.Sum(nil))[:32]
}

// writeField appends a length-prefixed field to h so the field concatenation is
// unambiguous — no field content can forge a boundary between fields.
func writeField(h hash.Hash, s string) {
	_, _ = fmt.Fprintf(h, "%d:", len(s))
	_, _ = h.Write([]byte(s))
}
