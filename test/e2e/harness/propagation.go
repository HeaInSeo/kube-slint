package harness

import (
	"fmt"
	"strings"

	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
)

// CheckStrictness evaluates the reliability of the measurement pipeline based on the StrictnessMode.
// It enforces the 'Strictness' propagation contract.
// CheckStrictness는 StrictnessMode를 기반으로 측정 파이프라인의 신뢰도를 평가합니다.
// 'Strictness' 전파 계약을 강제합니다.
func CheckStrictness(cfg SessionConfig, sum *summary.Summary) error {
	if sum == nil || sum.Reliability == nil {
		return fmt.Errorf("summary or reliability is nil")
	}

	mode := cfg.StrictnessMode
	if mode == "" {
		mode = "BestEffort" // default
	}

	rel := sum.Reliability

	// Check for any StatusBlock in results
	// 결과에 StatusBlock이 있는지 확인
	var blockReasons []string
	for _, res := range sum.Results {
		if res.Status == summary.StatusBlock {
			blockReasons = append(blockReasons, fmt.Sprintf("SLI %s blocked: %s", res.ID, res.Reason))
		}
	}

	if rel.BlockedReason != "" {
		blockReasons = append(blockReasons, fmt.Sprintf("pipeline blocked: %s", rel.BlockedReason))
	}

	isCollectionFailed := rel.CollectionStatus == "Failed" || rel.CollectionStatus == "Partial"
	isEvaluationFailed := rel.EvaluationStatus == "Failed" || rel.EvaluationStatus == "Partial"

	switch mode {
	case "BestEffort":
		// BestEffort does not promote pipeline failures to test errors.
		// BestEffort는 파이프라인 실패를 테스트 에러로 승격하지 않습니다.
		return nil

	case "StrictCollection":
		if isCollectionFailed {
			return fmt.Errorf("StrictCollection violation: collection status is %s, reasons: %s", rel.CollectionStatus, strings.Join(blockReasons, ", "))
		}

	case "StrictEvaluation":
		if isCollectionFailed || isEvaluationFailed {
			return fmt.Errorf("StrictEvaluation violation: evaluation status is %s, collection status is %s, reasons: %s", rel.EvaluationStatus, rel.CollectionStatus, strings.Join(blockReasons, ", "))
		}

	case "RequiredSLIs":
		// Example implementation for RequiredSLIs
		if isCollectionFailed || isEvaluationFailed {
			return fmt.Errorf("RequiredSLIs violation: pipeline failed, reasons: %s", strings.Join(blockReasons, ", "))
		}
		if len(rel.SkippedSLIs) > 0 {
			return fmt.Errorf("RequiredSLIs violation: some SLIs were skipped: %v", rel.SkippedSLIs)
		}

	default:
		// Check global block reasons if any unknown strictness is passed
		if len(blockReasons) > 0 {
			return fmt.Errorf("pipeline blocked (%s): %s", mode, strings.Join(blockReasons, ", "))
		}
	}

	// Always fail if there's explicit blocking reasons in strict modes
	if mode != "BestEffort" && len(blockReasons) > 0 {
		return fmt.Errorf("pipeline blocked: %s", strings.Join(blockReasons, ", "))
	}

	return nil
}

// CheckGating evaluates successfully computed SLIs against the gating policy.
// It enforces the 'GatingPolicy' propagation contract.
// CheckGating은 성공적으로 계산된 SLI를 게이팅 정책에 따라 평가합니다.
// 'GatingPolicy' 전파 계약을 강제합니다.
func CheckGating(cfg SessionConfig, sum *summary.Summary) error {
	if sum == nil {
		return fmt.Errorf("summary is nil")
	}

	gateOnLevel := cfg.GateOnLevel
	if gateOnLevel == "" || gateOnLevel == "none" {
		return nil
	}

	var gatingFailures []string

	for _, res := range sum.Results {
		// Ignore skipped or blocked results, those are handled by Strictness
		// 건너뛰거나 차단된 결과는 무시합니다. 이는 Strictness에서 처리됩니다.
		if res.Status == summary.StatusSkip || res.Status == summary.StatusBlock {
			continue
		}

		if gateOnLevel == "fail" && res.Status == summary.StatusFail {
			gatingFailures = append(gatingFailures, fmt.Sprintf("SLI %s failed: %s", res.ID, res.Reason))
		} else if gateOnLevel == "warn" && (res.Status == summary.StatusFail || res.Status == summary.StatusWarn) {
			gatingFailures = append(gatingFailures, fmt.Sprintf("SLI %s triggered level %s: %s", res.ID, res.Status, res.Reason))
		}
	}

	if len(gatingFailures) > 0 {
		return fmt.Errorf("GatingPolicy violation (%s): %s", gateOnLevel, strings.Join(gatingFailures, "; "))
	}

	return nil
}
