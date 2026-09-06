package summary

import (
	"testing"
	"time"
)

// KSL-T5: both the legacy and trust-correct measurement contracts are accepted,
// and any other version is rejected — a consumer never silently accepts an
// unknown contract.
func TestValidateSchemaVersion_ContractFence(t *testing.T) {
	base := Summary{GeneratedAt: time.Now()}
	for _, v := range []string{SchemaVersionLegacy, SchemaVersionTrust} {
		s := base
		s.SchemaVersion = v
		if err := ValidateSchemaVersion(s); err != nil {
			t.Fatalf("version %q must be accepted: %v", v, err)
		}
	}
	for _, v := range []string{"", "slo.v2", "slo.v5", "slint.summary.v4"} {
		s := base
		s.SchemaVersion = v
		if err := ValidateSchemaVersion(s); err == nil {
			t.Fatalf("version %q must be rejected", v)
		}
	}
}

func TestIsTrustCorrectContract(t *testing.T) {
	if IsTrustCorrectContract(Summary{SchemaVersion: SchemaVersionLegacy}) {
		t.Fatal("legacy contract must not be trust-correct")
	}
	if !IsTrustCorrectContract(Summary{SchemaVersion: SchemaVersionTrust}) {
		t.Fatal("trust contract must be trust-correct")
	}
}

// KSL-T4: a comparability identity is Complete only with all four coordinates,
// and Equal only when two complete identities match on every coordinate. A nil or
// incomplete identity is never comparable.
func TestComparability_CompleteAndEqual(t *testing.T) {
	full := &Comparability{SLIContractID: "c", SubjectID: "s", WindowID: "w", SourceConfigID: "cfg"}
	if !full.Complete() {
		t.Fatal("full identity must be complete")
	}
	var nilCmp *Comparability
	if nilCmp.Complete() {
		t.Fatal("nil identity must not be complete")
	}
	if (&Comparability{SLIContractID: "c"}).Complete() {
		t.Fatal("partial identity must not be complete")
	}
	if (&Comparability{SLIContractID: " ", SubjectID: "s", WindowID: "w", SourceConfigID: "cfg"}).Complete() {
		t.Fatal("a whitespace-only coordinate must not count as present")
	}
	if !full.Equal(&Comparability{SLIContractID: "c", SubjectID: "s", WindowID: "w", SourceConfigID: "cfg"}) {
		t.Fatal("identical full identities must be equal")
	}
	if full.Equal(nilCmp) {
		t.Fatal("a complete identity is never equal to nil")
	}
	if full.Equal(&Comparability{SLIContractID: "c", SubjectID: "s2", WindowID: "w", SourceConfigID: "cfg"}) {
		t.Fatal("a single-coordinate mismatch must not be equal")
	}
}
