package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"strings"

	"github.com/HeaInSeo/kube-slint/pkg/slo/spec"
	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
)

// protectedSchemaVersion returns the measurement contract version to stamp for a
// run: the trust-correct slo.v4 when a TrustContract is present, otherwise legacy
// slo.v3. Historical/unprotected output is never silently reinterpreted as v4.
func protectedSchemaVersion(cfg RunConfig) string {
	if cfg.TrustContract != nil {
		return summary.SchemaVersionTrust
	}
	return summary.SchemaVersion
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
			WindowID:       windowID(s),
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
// and its compute mode/aggregation/window and counter-reset policy. It deliberately
// excludes Judge/threshold/policy (which judge the value, not how it is measured),
// the input Alias (a cosmetic display name), and any run-scoped data.
func sliContractID(s spec.SLISpec) string {
	h := sha256.New()
	writeField(h, "kube-slint.sli.contract.v1")
	writeField(h, s.ID)
	writeField(h, s.Unit)
	writeField(h, s.Kind)
	writeField(h, string(s.Compute.Mode))
	writeField(h, string(s.Compute.OnCounterReset))
	writeField(h, fmt.Sprintf("inputs=%d", len(s.Inputs)))
	for _, in := range s.Inputs {
		writeField(h, in.Key)
	}
	return "slic-v1-" + hex.EncodeToString(h.Sum(nil))[:32]
}

// windowID derives a deterministic identity for the window/aggregation/query
// semantics of an SLI from its compute mode (which encodes point-vs-window and the
// aggregation, e.g. window_p95). It is never derived from StartedAt/FinishedAt
// elapsed runtime (KSL-E1 property 5).
func windowID(s spec.SLISpec) string {
	h := sha256.New()
	writeField(h, "kube-slint.sli.window.v1")
	writeField(h, string(s.Compute.Mode))
	return "win-v1-" + hex.EncodeToString(h.Sum(nil))[:32]
}

// writeField appends a length-prefixed field to h so the field concatenation is
// unambiguous — no field content can forge a boundary between fields.
func writeField(h hash.Hash, s string) {
	_, _ = fmt.Fprintf(h, "%d:", len(s))
	_, _ = h.Write([]byte(s))
}
