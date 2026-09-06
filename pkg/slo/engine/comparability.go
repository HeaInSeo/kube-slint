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
// Input identity is canonicalized per mode by canonicalInputKeys so that two specs
// which measure the same value share an SLIContractID while two that can measure
// differently do not.
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
	keys := canonicalInputKeys(mode, s.Inputs)
	writeField(h, fmt.Sprintf("inputs=%d", len(keys)))
	for _, k := range keys {
		writeField(h, k)
	}
	return "slic-v1-" + hex.EncodeToString(h.Sum(nil))[:32]
}

// canonicalInputKeys returns the input keys to hash for a mode's measurement
// identity, canonicalized so that reorderings/duplications that cannot change the
// measured value produce the same identity, while never merging inputs that can
// change it (which would risk a false comparison):
//
//   - window_min/max/p95/p99 select over the pooled samples (min/max compare;
//     percentile sorts a copy), so input ORDER cannot change the value → sort the
//     keys. Multiplicity is kept: a repeated input is pooled twice and shifts a
//     percentile.
//   - window_ratio's value uses only inputs[0] (numerator) and inputs[1]
//     (denominator); inputs[2:] are an unordered required-PRESENCE set, so the tail's
//     order and multiplicity — and any tail entry duplicating position 0/1 — cannot
//     change value or evidence → keep positions 0/1, then the sorted unique tail
//     excluding keys already required by 0/1.
//   - all other modes (single/start/end/delta point sums, window_avg) combine inputs
//     by float addition, which is not associative, so input order is part of the
//     value → preserve it exactly.
func canonicalInputKeys(mode spec.ComputeMode, inputs []spec.MetricRef) []string {
	keys := make([]string, len(inputs))
	for i, in := range inputs {
		keys[i] = in.Key
	}
	switch mode {
	case spec.ComputeWindowMin, spec.ComputeWindowMax, spec.ComputeWindowP95, spec.ComputeWindowP99:
		sort.Strings(keys)
		return keys
	case spec.ComputeWindowRatio:
		if len(keys) <= 2 {
			return keys
		}
		seen := map[string]bool{keys[0]: true, keys[1]: true}
		tail := make([]string, 0, len(keys)-2)
		for _, k := range keys[2:] {
			if seen[k] {
				continue
			}
			seen[k] = true
			tail = append(tail, k)
		}
		sort.Strings(tail)
		return append(keys[:2:2], tail...)
	default:
		return keys
	}
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
