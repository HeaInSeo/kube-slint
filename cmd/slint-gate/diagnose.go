package main

import (
	"fmt"
	"strings"

	"github.com/HeaInSeo/kube-slint/internal/gate"
)

// diagEntry는 하나의 reason 코드에 대한 진단 항목이다.
type diagEntry struct {
	summary string
	hints   []string
}

// diagMessages는 reason 코드를 사람이 읽을 수 있는 진단 메시지로 변환한다.
var diagMessages = map[string]diagEntry{
	"MEASUREMENT_INPUT_MISSING": {
		summary: "sli-summary.json 파일이 없거나 비어 있습니다.",
		hints: []string{
			"E2E 테스트(sess.End(ctx)) 실행 후에 slint-gate를 실행하세요.",
			"SessionConfig.ArtifactsDir 경로가 올바른지 확인하세요.",
			"E2E 로그에서 'fetch failed' 메시지를 확인하세요.",
			"RBAC 확인:\n    kubectl auth can-i create pods" +
				" --as=system:serviceaccount:<ns>:<sa> -n <ns>",
		},
	},
	"MEASUREMENT_INPUT_CORRUPT": {
		summary: "sli-summary.json 파일을 파싱할 수 없습니다.",
		hints: []string{
			"파일이 완전한 JSON인지 확인하세요: cat artifacts/sli-summary.json | python3 -m json.tool",
			"E2E 테스트가 중간에 중단되어 파일이 불완전하게 쓰였을 수 있습니다.",
		},
	},
	"POLICY_MISSING": {
		summary: ".slint/policy.yaml 파일이 없습니다.",
		hints: []string{
			"다음 명령으로 기본 policy.yaml을 생성하세요:\n    slint-gate init",
			"--policy 플래그로 경로를 직접 지정할 수도 있습니다:\n    slint-gate --policy path/to/policy.yaml",
		},
	},
	"POLICY_INVALID": {
		summary: "policy.yaml 파일이 올바르지 않습니다.",
		hints: []string{
			"YAML 문법 오류가 없는지 확인하세요: cat .slint/policy.yaml",
			"operator 필드에 지원되지 않는 연산자(예: !=)가 있는지 확인하세요.",
			"지원 연산자: <=, >=, <, >, ==",
		},
	},
	"THRESHOLD_MISS": {
		summary: "하나 이상의 임계값 조건을 위반했습니다.",
		hints: []string{
			"slint-gate-summary.json의 checks 항목에서 fail 상태인 항목을 확인하세요.",
			"임계값을 조정하거나 오퍼레이터 동작을 검토하세요.",
		},
	},
	"REGRESSION_DETECTED": {
		summary: "baseline 대비 지표가 허용 오차를 초과했습니다.",
		hints: []string{
			"slint-gate-summary.json의 regression 검사 항목을 확인하세요.",
			"정상적인 변경이라면 baseline을 업데이트하세요:\n    make baseline-update-prepare BASELINE_SUMMARY=artifacts/sli-summary.json",
			"일시적인 변동이라면 policy.yaml의 tolerance_percent를 조정하세요.",
		},
	},
	"BASELINE_ABSENT_FIRST_RUN": {
		summary: "baseline이 없습니다 (첫 실행). 회귀 비교가 생략됩니다.",
		hints: []string{
			"이것은 경고입니다. 첫 실행에서는 정상입니다.",
			"baseline을 저장하려면:\n    make baseline-update-prepare BASELINE_SUMMARY=artifacts/sli-summary.json",
			"baseline 없이 계속하려면 policy.yaml에서 regression.enabled: false로 설정하세요.",
		},
	},
	"BASELINE_UNAVAILABLE": {
		summary: "--baseline으로 지정한 파일을 찾을 수 없습니다.",
		hints: []string{
			"--baseline 플래그에 전달한 경로가 올바른지 확인하세요.",
			"baseline이 없으면 --baseline 플래그를 생략하세요.",
		},
	},
	"BASELINE_CORRUPT": {
		summary: "baseline 파일을 파싱할 수 없습니다.",
		hints: []string{
			"baseline 파일이 완전한 JSON인지 확인하세요.",
			"baseline을 새로 갱신하세요:\n    make baseline-update-prepare BASELINE_SUMMARY=artifacts/sli-summary.json",
		},
	},
	"RELIABILITY_INSUFFICIENT": {
		summary: "메트릭 수집 신뢰도가 policy.yaml의 최소 요구 수준 미만입니다.",
		hints: []string{
			"E2E 로그에서 fetch 오류가 있는지 확인하세요.",
			"허용하려면 policy.yaml에서 reliability.required: false로 설정하세요.",
		},
	},
}

// printDiagnostics는 gate 결과가 PASS가 아닐 때 진단 메시지를 stdout에 출력한다.
func printDiagnostics(result *gate.Summary) {
	if result.GateResult == gate.GatePass {
		return
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "\n── Diagnostics ─────────────────────────────────────────────────────────\n")
	fmt.Fprintf(&sb, "Gate Result: %s\n", result.GateResult)

	if len(result.Reasons) == 0 {
		fmt.Fprintf(&sb, "────────────────────────────────────────────────────────────────────────\n")
		fmt.Print(sb.String())
		return
	}

	for _, reason := range result.Reasons {
		entry, ok := diagMessages[reason]
		if !ok {
			fmt.Fprintf(&sb, "\n[%s]\n  (추가 진단 정보 없음)\n", reason)
			continue
		}
		fmt.Fprintf(&sb, "\n[%s]\n  %s\n", reason, entry.summary)
		for _, hint := range entry.hints {
			indented := "  → " + strings.ReplaceAll(hint, "\n", "\n    ")
			fmt.Fprintf(&sb, "%s\n", indented)
		}
	}
	fmt.Fprintf(&sb, "────────────────────────────────────────────────────────────────────────\n")
	fmt.Print(sb.String())
}
