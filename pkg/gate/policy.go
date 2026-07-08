package gate

import (
	"fmt"
	"math"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// knownPolicyKeys is the set of top-level keys that policy.yaml supports.
var knownPolicyKeys = map[string]bool{
	"schema_version":  true,
	"thresholds":      true,
	"regression":      true,
	"reliability":     true,
	"fail_on":         true,
	"promote_to_fail": true,
}

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

	var warnings []string
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err == nil {
		warnings = collectUnknownPolicyKeys(&doc)
	}

	var p Policy
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, policyInvalid, warnings
	}
	if err := validatePolicy(p); err != nil {
		warnings = append(warnings, err.Error())
		return nil, policyInvalid, warnings
	}
	if len(p.FailOn) > 0 {
		warnings = append(warnings,
			"policy.yaml: 'fail_on' is deprecated; use 'promote_to_fail' instead (both are honored during the deprecation window)")
	}
	return &p, policyOK, warnings
}

func validatePolicy(p Policy) error {
	if strings.TrimSpace(p.SchemaVersion) != "slint.policy.v1" {
		return fmt.Errorf("unsupported schema_version %q (want slint.policy.v1)", p.SchemaVersion)
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
	seenNames := map[string]bool{}
	for _, rule := range p.Thresholds {
		if math.IsNaN(rule.Value) {
			return fmt.Errorf("threshold %q has a NaN value", rule.Name)
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
	if p.Regression.Enabled && p.Regression.TolerancePercent < 0 {
		return fmt.Errorf("regression.tolerance_percent must not be negative (got %v)", p.Regression.TolerancePercent)
	}
	return nil
}

// collectUnknownPolicyKeys walks the top-level mapping node and returns
// warning messages for any key not in knownPolicyKeys.
func collectUnknownPolicyKeys(doc *yaml.Node) []string {
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil
	}
	var warnings []string
	for i := 0; i+1 < len(root.Content); i += 2 {
		key := root.Content[i].Value
		if !knownPolicyKeys[key] {
			warnings = append(warnings,
				fmt.Sprintf("unknown field %q in policy.yaml (line %d) — ignored; supported fields: schema_version, thresholds, regression, reliability, fail_on (deprecated), promote_to_fail",
					key, root.Content[i].Line))
		}
	}
	return warnings
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
	}
	return result
}

var allowedPromotionValues = map[string]bool{
	"threshold_miss":      true,
	"regression_detected": true,
}

func normalizePromotionValue(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}
