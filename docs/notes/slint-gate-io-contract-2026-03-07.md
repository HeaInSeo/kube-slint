# slint-gate I/O contract (draft, 2026-03-07)

## A) Policy file path proposal

Primary recommendation (proposed):
- `.slint/policy.yaml`

Reason:
- repository root에서 명확히 분리된 전용 정책 경로
- 개발자 로컬/CI/consumer repo에서 동일한 상대 경로로 참조하기 쉬움
- 기존 `docs/*`(사람용)와 분리되어 자동화 입력 소스로 쓰기 적합

Alternatives (draft):
- `config/slint/policy.yaml`
- `test/e2e/slint/policy.yaml`

Notes:
- 이번 단계에서는 경로를 "권장안(primary recommendation)"으로 제시하며, 최종 확정은 구현 단계에서 결정.

## B) Minimum viable policy schema (draft)

Purpose:
- threshold / regression / reliability / first-run / baseline / fail semantics를 최소 필드로 표현.

```yaml
schema_version: "slint.policy.v1"
metadata:
  name: "default-guardrail-policy"
  owner: "platform-team"

thresholds:
  - name: "reconcile_error_ratio_max"
    metric: "reconcile_error_ratio"
    operator: "<="
    value: 0.02
    severity: "fail"

regression:
  enabled: true
  mode: "percent_delta"
  tolerance_percent: 5

reliability:
  required: false
  min_level: "partial"

first_run:
  default_result: "warn"
  evaluate_thresholds: true
  evaluate_regression: false

baseline:
  required: false
  on_unavailable: "warn"
  on_corrupt: "no_grade"

fail_on:
  - "threshold_miss"
  - "regression_detected"
```

Schema notes:
- `schema_version`은 정책 호환성 식별용.
- `thresholds`는 absolute gate 축.
- `regression`은 baseline 존재 시 1급 gate 축.
- `reliability`는 측정 신뢰도 기반의 보조/강제 정책 선택축.
- `first_run`, `baseline`은 baseline 유무/손상 시 기본 처리 정책.
- `fail_on`은 CI fail 승격 기준을 명시.

## C) `slint-gate-summary.json` output contract (draft)

Minimum fields:
- `schema_version`: 출력 스키마 버전 (예: `slint.gate.v1`)
- `gate_result`: `PASS | WARN | FAIL | NO_GRADE`
- `evaluation_status`: `evaluated | partially_evaluated | not_evaluated`
- `measurement_status`: `ok | missing | corrupt | insufficient`
- `baseline_status`: `present | absent_first_run | unavailable | corrupt`
- `policy_status`: `ok | missing | invalid`
- `reasons`: 판정 이유 코드/메시지 목록
- `evaluated_at`: 평가 시각(UTC)
- `input_refs`: 입력 파일/아티팩트 참조 경로
- `checks`: threshold/regression/reliability 세부 결과
- `overall_message`: 사람용 한 줄 요약

Proposed shape:

```json
{
  "schema_version": "slint.gate.v1",
  "gate_result": "WARN",
  "evaluation_status": "partially_evaluated",
  "measurement_status": "ok",
  "baseline_status": "absent_first_run",
  "policy_status": "ok",
  "reasons": ["baseline_missing_first_run"],
  "evaluated_at": "2026-03-07T12:00:00Z",
  "input_refs": {
    "measurement_summary": "artifacts/sli-summary.json",
    "policy_file": ".slint/policy.yaml",
    "baseline_file": null
  },
  "checks": [],
  "overall_message": "Thresholds passed; regression skipped on first run."
}
```

## D) `checks` structure proposal

Recommended `checks[]` item fields:
- `name`
- `category` (`threshold | regression | reliability`)
- `status` (`pass | warn | fail | no_grade`)
- `metric`
- `observed`
- `expected`
- `message`

Example:

```json
{
  "name": "reconcile_error_ratio_max",
  "category": "threshold",
  "status": "pass",
  "metric": "reconcile_error_ratio",
  "observed": 0.013,
  "expected": "<= 0.02",
  "message": "within threshold"
}
```

Why needed:
- machine-readable 판정 재사용 가능
- Actions summary/PR comment 생성에 직접 활용 가능
- 추후 리포트/대시보드 연계 시 구조 변경 비용 감소

## E) Status axis definitions

- `gate_result`: 최종 gate 판단
- `evaluation_status`: 평가 수행 완결성
- `measurement_status`: 측정 입력의 가용성/무결성
- `baseline_status`: baseline 비교 축의 가용 상태
- `policy_status`: 정책 구성의 유효 상태

Why split:
- measurement failure와 policy violation을 같은 층에서 섞지 않기 위해
- "측정 불가"와 "정책 위반"을 구분해 CI/요약 메시지의 의미를 보존하기 위해

## F) first-run / baseline example summaries

### 1) first-run, threshold ok, no baseline
- `gate_result`: `WARN` (default proposal)
- `evaluation_status`: `partially_evaluated`
- `baseline_status`: `absent_first_run`
- `reasons`: `["baseline_missing_first_run"]`

### 2) baseline present, regression detected
- `gate_result`: `FAIL`
- `evaluation_status`: `evaluated`
- `baseline_status`: `present`
- `reasons`: `["regression_detected"]`

### 3) baseline corrupt or unavailable
- `gate_result`: `NO_GRADE` (comparison-required 정책 기준)
- `evaluation_status`: `partially_evaluated`
- `baseline_status`: `corrupt` or `unavailable`
- `reasons`: `["baseline_unreadable"]` or `["baseline_unavailable"]`

## G) CI fail semantics (proposed default)

- `FAIL` -> CI non-zero 후보 (정책 위반)
- `WARN` -> 기본 CI 통과, summary로 노출
- `NO_GRADE` -> 기본 CI 통과, summary로 노출

Open question (planned):
- `policy_status=missing|invalid`를 즉시 infra/config failure로 승격해 CI fail할지 여부
- 초기 단계에서는 `NO_GRADE` 유지 후, 운영 정책 성숙도에 맞춰 강화 검토

## H) hello-operator connection

For hello-operator:
- first-run: absolute threshold 중심 정책으로 시작
- baseline 확보 이후: regression rule 활성화

Target UX (ko + Tilt inner-loop + CI):
- 코드 수정 후 operational SLI 회귀를 조기에 감지
- 개발자가 correctness 문제와 guardrail 정책 위반을 구분해서 대응 가능

## Status

- This is a proposed/draft I/O contract.
- No workflow/code/baseline storage implementation is included in this step.
