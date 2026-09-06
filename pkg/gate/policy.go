package gate

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func loadPolicy(path string) (*Policy, string, []string) {
	if path == "" {
		return nil, policyMissing, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, policyMissing, nil
		}
		// A non-NotExist error (permission denied, EISDIR, etc.) is an OS/IO
		// failure, not a YAML content problem — surface the real error via
		// the same warnings channel diagnose.go's POLICY_INVALID hints read,
		// instead of leaving the caller to guess it's a syntax issue.
		return nil, policyInvalid, []string{fmt.Sprintf("could not read policy file %s: %v", path, err)}
	}

	// KSL-T1: whole-policy strict validity before any protected grade. An
	// unknown key at ANY semantic level, a duplicate mapping key, or a trailing
	// second YAML document makes the policy invalid (fail-closed) rather than a
	// silently-ignored or warn-only condition. KnownFields(true) rejects unknown
	// fields recursively (top-level and nested); yaml.v3 rejects duplicate
	// mapping keys on its own. An invalid-but-readable policy must never yield a
	// protected grade, so the caller maps policyInvalid to NO_GRADE.
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var p Policy
	if err := dec.Decode(&p); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, policyInvalid, []string{"policy.yaml is empty"}
		}
		return nil, policyInvalid, []string{fmt.Sprintf("policy.yaml is invalid: %v", err)}
	}
	// Exactly one document: a trailing second YAML document is rejected so a
	// policy cannot smuggle in a second, unevaluated definition.
	if err := dec.Decode(new(yaml.Node)); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, policyInvalid, []string{"policy.yaml must contain exactly one YAML document"}
		}
		return nil, policyInvalid, []string{fmt.Sprintf("policy.yaml is invalid after the first document: %v", err)}
	}

	if err := validatePolicy(p); err != nil {
		return nil, policyInvalid, []string{err.Error()}
	}
	var warnings []string
	if len(p.FailOn) > 0 {
		warnings = append(warnings,
			"policy.yaml: 'fail_on' is deprecated; use 'promote_to_fail' instead (both are honored during the deprecation window)")
	}
	return &p, policyOK, warnings
}

// Policy-contract versions (KSL-T5 / packet §4). policyContractTrust
// ("slint.policy.v2") is the trust-correct policy contract required to consume
// the trust-correct measurement contract for a protected baseline comparison;
// policyContractLegacy ("slint.policy.v1") is still accepted and evaluated but is
// explicitly legacy — it never silently acquires the v2 trust-correct semantics.
const (
	policyContractLegacy = "slint.policy.v1"
	policyContractTrust  = "slint.policy.v2"
)

// policyIsTrustCorrect reports whether the policy declares the trust-correct
// contract (slint.policy.v2).
func policyIsTrustCorrect(p *Policy) bool {
	return strings.TrimSpace(p.SchemaVersion) == policyContractTrust
}

func validatePolicy(p Policy) error {
	sv := strings.TrimSpace(p.SchemaVersion)
	if sv != policyContractLegacy && sv != policyContractTrust {
		return fmt.Errorf("unsupported schema_version %q (want %s or %s)",
			p.SchemaVersion, policyContractLegacy, policyContractTrust)
	}
	for _, item := range p.FailOn {
		v := normalizePromotionValue(item)
		if v == "" {
			continue
		}
		if !allowedPromotionValues[v] {
			return fmt.Errorf("unsupported fail_on value %q", item)
		}
	}
	for _, item := range p.PromoteToFail {
		v := normalizePromotionValue(item)
		if v == "" {
			continue
		}
		if !allowedPromotionValues[v] {
			return fmt.Errorf("unsupported promote_to_fail value %q", item)
		}
	}
	minLevel := strings.ToLower(strings.TrimSpace(p.Reliability.MinLevel))
	if minLevel != "" && minLevel != "partial" && minLevel != "complete" {
		return fmt.Errorf("unsupported reliability.min_level %q", p.Reliability.MinLevel)
	}
	if err := validateThresholds(p.Thresholds); err != nil {
		return err
	}
	if p.Regression.Enabled && p.Regression.TolerancePercent < 0 {
		return fmt.Errorf("regression.tolerance_percent must not be negative (got %v)", p.Regression.TolerancePercent)
	}
	return nil
}

// validateThresholds enforces KSL-T1 strict validity for threshold rules: required
// semantic coordinates present, a supported operator, no NaN value, and unique
// non-empty identities — all before any evaluation.
func validateThresholds(rules []ThresholdRule) error {
	seenNames := map[string]bool{}
	for _, rule := range rules {
		if math.IsNaN(rule.Value) {
			return fmt.Errorf("threshold %q has a NaN value", rule.Name)
		}
		if strings.TrimSpace(rule.Metric) == "" {
			return fmt.Errorf("threshold %q has an empty metric (a required coordinate)", rule.Name)
		}
		op := strings.TrimSpace(rule.Operator)
		if op == "" {
			return fmt.Errorf("threshold %q has an empty operator (a required coordinate)", rule.Name)
		}
		if !supportedOperators[op] {
			return fmt.Errorf("threshold %q has an unsupported operator %q", rule.Name, rule.Operator)
		}
		name := strings.TrimSpace(rule.Name)
		if name == "" {
			continue // empty names are allowed and auto-assigned ("unnamed-threshold"); see evalThreshold
		}
		if seenNames[name] {
			return fmt.Errorf("duplicate threshold name %q", name)
		}
		seenNames[name] = true
	}
	return nil
}

func makePromotionSet(policy *Policy) map[string]bool {
	result := map[string]bool{}
	// FailOn (deprecated) and PromoteToFail are unioned: whichever field(s)
	// are populated take effect, so a policy authored under either name
	// behaves identically.
	for _, item := range policy.FailOn {
		item = normalizePromotionValue(item)
		if item != "" {
			result[item] = true
		}
	}
	for _, item := range policy.PromoteToFail {
		item = normalizePromotionValue(item)
		if item != "" {
			result[item] = true
		}
	}
	if len(result) == 0 {
		result["threshold_miss"] = true
		result["regression_detected"] = true
		result["coverage_gap"] = true
	}
	return result
}

var allowedPromotionValues = map[string]bool{
	"threshold_miss":      true,
	"regression_detected": true,
	"coverage_gap":        true,
}

// supportedOperators is the set of threshold operators CompareOp understands.
// Policy validation (KSL-T1) rejects any threshold whose operator is not in this
// set before evaluation, so an unsupported operator is whole-policy invalidity
// rather than a per-check no_grade discovered mid-evaluation.
var supportedOperators = map[string]bool{
	"<=": true, "=<": true,
	">=": true, "=>": true,
	"<": true, ">": true,
	"==": true, "=": true,
}

func normalizePromotionValue(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}
