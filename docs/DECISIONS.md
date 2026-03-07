# kube-slint Decision Log

This file records architecture/product-direction decisions that define the project contract.

---

## D-001: kube-slint identity = shift-left operational quality guardrail

- Date: 2026-03-07
- Status: Accepted
- Decision:
  - kube-slint는 "operator correctness test framework"가 아니라,
    **operator 개발 단계에서 operational SLI를 lint-style로 적용하여 성능/신뢰성/회귀를 조기에 차단하는 shift-left quality guardrail**로 정의한다.
- Rationale:
  - 현재 구현은 library/harness foundation이 안정적이며, 이를 guardrail 제품 정체성으로 명확히 고정해야 문서/CI/소비자 UX가 일관된다.

## D-002: measurement failure != test failure; policy violation may fail CI

- Date: 2026-03-07
- Status: Accepted
- Decision:
  - 계측 실패(measurement failure)는 기본적으로 테스트 실패와 동일시하지 않는다.
  - 정책 위반(policy violation: absolute threshold miss, regression vs baseline)은 CI 실패로 승격될 수 있다.
- Rationale:
  - 비침투 계측 철학(best-effort measurement)을 유지하면서도, 실제 품질 게이트는 policy 레이어에서 강제해야 한다.

## D-003: measurement mode taxonomy is first-class

- Date: 2026-03-07
- Status: Accepted
- Decision:
  - 다음 3개 모드를 1급 개념(first-class)으로 유지한다.
    - `InsideSnapshot` (default)
    - `InsideAnnotation` (precise / semantic-boundary)
    - `OutsideSnapshot` (environment-specific)
- Rationale:
  - 모드 선택은 계측 책임 경계와 UX를 좌우하므로 문서/설정/리포팅에서 숨은 구현 디테일이 아니라 명시적 계약이어야 한다.

## D-004: regression comparison is first-class gate

- Date: 2026-03-07
- Status: Accepted
- Decision:
  - baseline 대비 회귀 비교(regression vs baseline)를 절대 임계치와 동급의 policy gate로 취급한다.
- Rationale:
  - shift-left guardrail의 핵심은 "지금 절대값"뿐 아니라 "이전 대비 악화"를 조기에 차단하는 것이다.

## D-005: hello-operator is canonical consumer DX validation repo

- Date: 2026-03-07
- Status: Accepted
- Decision:
  - `hello-operator`를 kube-slint 소비자 DX 검증의 canonical repo로 고정한다.
  - 향후 ko+tilt inner-loop 검증은 해당 저장소 기준으로 수행한다.
- Rationale:
  - 소비자 관점 검증 기준점이 단일화되어야 온보딩/회귀/문서 검증의 신뢰도가 높아진다.

## D-006: Guardrail evaluation is separate from correctness testing

- Date: 2026-03-07
- Status: Accepted
- Decision:
  - `test/lint/mock-e2e`와 `slint-gate`는 역할이 다르다.
  - correctness testing은 구현 정합성을 검증하고,
  - guardrail evaluation은 정책 위반(절대 임계치 미달, baseline 대비 회귀)을 별도 gate job에서 판정한다.
- Rationale:
  - measurement failure와 policy failure를 분리해야 비침투/best-effort 철학을 유지하면서도 CI 품질 게이트를 명확히 운영할 수 있다.
