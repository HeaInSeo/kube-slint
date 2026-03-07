# Phase 6-b Guardrail Alignment Notes
*Date: 2026-03-07*

## 1) README / README(Kor) follow-up points (small-diff guidance)

- 목적 문구를 "test framework" 중심에서 "shift-left lint-style operational SLI guardrail" 중심으로 전환.
- 핵심 계약을 상단에 명시:
  - measurement failure != test failure
  - policy violation may fail CI
- measurement modes를 1급 taxonomy로 명시:
  - InsideSnapshot (default)
  - InsideAnnotation (precise / semantic-boundary)
  - OutsideSnapshot (environment-specific)
- regression comparison(baseline 대비 악화 차단)을 policy gate 핵심 항목으로 명시.
- `hello-operator`를 canonical consumer DX validation repo로 명시.

## 2) GitHub Actions direction memo (documentation-only)

### A. `slint-gate`
- 목적:
  - policy 위반(absolute miss / regression)을 CI 실패로 승격.
  - measurement 실패는 리포트에 남기되 기본적으로 테스트 실패와 분리.
- 출력:
  - policy 판단 결과(FAIL/WARN/PASS)
  - measurement reliability 요약(Complete/Partial/Failed)

### B. `roadmap-status`
- 목적:
  - 현재 stage(예: Phase 6-b/6-c/6-d) 및 계약 충족 상태를 PR/CI에서 가시화.
- 출력:
  - 현재 Stage, DoD 체크 상태, open gap(회귀게이트/가시성/consumer DX)

### C. `baseline-update`
- 목적:
  - baseline 갱신을 일반 PR과 분리하고 승인 기반으로 관리.
- 정책:
  - baseline update는 명시적 intent(라벨/수동 트리거)와 diff 근거를 남겨야 함.

## 3) Guardrail message checklist

- "테스트 프레임워크"가 아니라 "테스트와 구별되는 품질 가드레일"이라는 문구를 반복적으로 고정.
- measurement 결과와 policy verdict를 같은 의미로 혼용하지 않음.
- 기존 non-invasive / best-effort 철학을 유지.

