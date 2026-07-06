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

## D-007: Automation status source is docs/project-status.yaml

- Date: 2026-03-07
- Status: Accepted
- Decision:
  - 자동화(workflow/summary job)가 읽는 machine-readable 상태 소스는 `docs/project-status.yaml` 단일 경로로 고정한다.
  - prose 문서(`docs/PROGRESS_LOG.md` 등) 파싱은 자동화 입력으로 사용하지 않는다.
- Rationale:
  - 마크다운 서술 문서는 사람용 맥락 기록에 최적화되어 있고, 자동화 안정성은 고정 스키마 YAML에서 보장하는 편이 안전하다.

## D-008: slint-gate is a separate policy evaluation layer over measurement outputs

- Date: 2026-03-07
- Status: Accepted
- Decision:
  - `slint-gate`는 correctness test를 대체하지 않으며, 계측 결과물 위에서 정책을 평가하는 별도 레이어로 둔다.
  - gate 판정은 `PASS | WARN | FAIL | NO_GRADE`를 기본 enum으로 사용한다.
- Rationale:
  - measurement failure와 policy violation을 분리하여, 비침투/best-effort 철학과 CI 품질 게이트를 동시에 유지하기 위함.

## D-009: Baseline comparison is optional on first-run, first-class when baseline exists

- Date: 2026-03-07
- Status: Accepted
- Decision:
  - baseline이 없는 first-run에서는 regression 비교를 강제하지 않는다.
  - baseline이 존재하면 regression 비교를 1급 gate 축으로 평가한다.
- Rationale:
  - 초기 도입 마찰을 낮추되, baseline 확보 이후에는 회귀 차단을 핵심 gate로 운영하기 위함.

## D-010: Primary policy input path recommendation is .slint/policy.yaml (proposed)

- Date: 2026-03-07
- Status: Accepted (proposed contract)
- Decision:
  - `slint-gate` 정책 파일 기본 경로 권장안은 `.slint/policy.yaml`로 둔다.
  - 대체 경로(`config/slint/policy.yaml`, `test/e2e/slint/policy.yaml`)는 호환 가능한 옵션으로 유지한다.
- Rationale:
  - 자동화 입력 소스와 사람용 문서를 분리하고, consumer repo에서도 동일한 패턴으로 재사용하기 쉽다.

## D-011: slint-gate-summary.json is the machine-readable output contract for gate evaluation

- Date: 2026-03-07
- Status: Accepted (implemented)
- Decision:
  - gate 결과의 machine-readable 출력 계약은 `slint-gate-summary.json`으로 정의한다.
  - 최소 필드(`gate_result`, `evaluation_status`, `measurement_status`, `baseline_status`, `policy_status`, `checks` 등)를 유지한다.
- Rationale:
  - Actions summary, PR 코멘트, 후속 리포팅 경로가 동일한 gate 결과 구조를 재사용할 수 있다.

## D-012: slint-gate CLI is implemented in Go (cmd/slint-gate); Python prototype removed

- Date: 2026-04-30 (updated 2026-05-11)
- Status: Accepted, Completed
- Decision:
  - `hack/slint_gate.py` (Python + pyyaml)는 삭제되었다. 운영 gate 경로는 `cmd/slint-gate` Go 바이너리만 사용한다.
  - gate 평가 로직은 `internal/gate` 패키지로 캡슐화하며, CLI는 `cmd/slint-gate/main.go`에서만 flag 파싱 및 출력을 담당한다.
  - `hack/prepare-baseline-update.sh`도 `go run ./cmd/slint-gate` + `jq` 기반으로 재작성되었다.
- Rationale:
  - Go 단일 언어 스택으로 통합하여 Python 런타임/pyyaml 의존을 완전 제거한다.
  - `internal/gate` 단위 테스트로 게이트 로직의 회귀를 방어한다.
  - `--github-step-summary` 플래그로 Actions step summary 렌더링을 Go 바이너리 내부로 흡수한다.

## D-013: pkg/slint owns the consumer-facing session API

- Date: 2026-06-27
- Status: Accepted, Implemented
- Decision:
  - `pkg/slint`가 `Session`, `SessionConfig`, `NewSession`, 기본 SLI specs, discovery, propagation, cleanup, and curlpod-backed session implementation의 소비자용 구현을 직접 소유한다.
  - `test/e2e/harness`는 기존 내부/테스트 import 경로 호환성을 위한 얇은 wrapper로만 유지한다.
- Rationale:
  - 공개 라이브러리 API가 `test/e2e` 경로를 import하는 구조는 consumer 관점에서 미완성처럼 보이며, 공모전/오픈소스 온보딩 첫인상에 불리하다.
  - 구현 소유권을 `pkg/slint`로 올려도 measurement failure와 policy gate 분리 원칙은 바뀌지 않는다.

## D-014: Post-RC hardening prioritizes secret containment and conservative gate semantics

- Date: 2026-07-02
- Status: Accepted
- Decision:
  - Post-RC hardening의 최우선 순위는 curlpod bearer token 노출 제거, command log redaction, namespace-scoped RBAC, 그리고 `NO_GRADE`가 CI gate에서 명확히 다뤄지는 보수적 판정 흐름이다.
  - 계측 실패는 D-002에 따라 테스트 실패와 직접 동일시하지 않는다. 대신 summary/gate 모델에서 `measurement_status=insufficient` 또는 `gate_result=NO_GRADE`로 드러내고, CI 실패 여부는 `fail-on` 정책이 결정한다.
  - 알 수 없는 policy/CLI enum 값은 조용히 무시하지 않고 invalid input으로 처리한다.
- Rationale:
  - kube-slint는 운영 SLI guardrail이므로 token 노출, 과도한 RBAC, 측정 불충분의 PASS 오인 가능성은 correctness feature보다 먼저 줄여야 한다.
  - 동시에 best-effort measurement 철학을 유지하려면 계측 실패와 policy violation의 책임 경계를 계속 분리해야 한다.

## D-015: Quality roadmap contracts are CI-guarded planning inputs

- Date: 2026-07-04
- Status: Accepted
- Decision:
  - 8 -> 9 -> 10 quality roadmap work is tracked as a non-runtime planning and
    guardrail workstream unless an implementation task explicitly says
    otherwise.
  - The quality roadmap contracts live in `docs/quality-roadmap.md`,
    `docs/quality-roadmap-implementation-handoff.md`,
    `docs/security-model.md`, `docs/gate-contract.md`,
    `docs/test-strategy.md`, and `docs/release-devex-plan.md`.
  - `hack/quality-guardrails.sh` and
    `.github/workflows/quality-guardrails.yml` are accepted as CI-backed drift
    detection for identity, security, RBAC, schema, and gate-contract wording.
  - These guardrails may check that proposed-contract documents exist and stay
    aligned, but they must not claim that unimplemented runtime behavior has
    shipped.
  - Runtime behavior changes still require normal implementation work, tests,
    and source-of-truth updates.
- Rationale:
  - The quality roadmap contains many high-impact security and CI contracts.
    Capturing them only as prose would let implementation drift or stale docs
    re-enter the repo.
  - CI-backed drift detection is appropriate for accepted identity/security
    contracts, while future behavior must remain clearly labeled as proposed
    until implemented.

## D-016: SLI Gate Onboarding UX ships as a guided CLI loop, not an invented one

- Date: 2026-07-07
- Status: Accepted
- Decision:
  - The onboarding CLI surface (`init --profile`, `inspect`, `recommend-policy`,
    `baseline approve/diff/merge`, `ci github-actions`, `quickstart`) follows
    "measure -> explain -> recommend -> approve -> CI" per
    `docs/sli-gate-onboarding-ux.md`, built across Sprint 1-6.
  - `policy.fail_on`/CLI `--fail-on`/action `fail-on` are renamed to
    `promote_to_fail`/`--exit-on`/`exit-on` (they controlled two different
    layers despite the shared "fail" wording) with the old names kept as
    working, deprecated aliases (dual-support, not a breaking rename) since
    they already shipped in tagged releases.
  - The `kubebuilder-operator` profile's SLI candidates are tiered
    (`core`/`noisy`/`informational`), and only `--strictness` governs the
    `noisy` tier; a candidate is never given a fabricated threshold it can't
    principled support (see the `informational` tier for raw activity
    counters).
  - No second built-in profile (e.g. `dataplane-service`) was added, since no
    real SLI spec/collector exists for one in this codebase; local custom
    profile files (`--profile-file`) are the extensibility path instead.
  - Sprint 6's "interactive wizard" shipped as a non-interactive `quickstart`
    status command instead — a real stdin-prompted CLI is a different kind
    of engineering risk (TTY detection, non-interactive-CI handling, harder
    to test) than anything else in this tool, and the scoping question went
    unanswered under deadline pressure.
- Rationale:
  - Every naming/scope decision in this roadmap follows the same rule this
    project already established with `Dangerously*` options: prefer a
    visible, honest name or an explicit deferral over a rename that breaks
    existing callers, or a feature that pretends to know something (a safe
    threshold, a real second profile) it doesn't actually have grounds for.

## D-017: Custom Semgrep guardrails are blocking CI, not advisory-only

- Date: 2026-07-07
- Status: Accepted
- Decision:
  - The 6 rules `docs/security-model.md`'s "Static Guardrail Plan" already
    named (and left unimplemented) are implemented in `.semgrep/rules/`,
    each with a paired positive/negative Go fixture, and enabled as
    blocking CI (`.github/workflows/semgrep.yml`) rather than rolled out
    advisory-first.
  - This was possible without an advisory phase because the real codebase
    was scanned and made fully compliant in the same change: two
    already-accepted patterns (the `--output`/`--baseline`
    overwrite-refusal checks; `sweep.go`'s label-filtered-then-delete-by-name
    cleanup) got a bare `// nosemgrep` plus a reason comment, and
    `pkg/kubeutil.ApplyClusterRoleBinding` (documented dead/test-only code)
    is excluded wholesale via `.semgrepignore`.
  - Inline suppressions use bare `// nosemgrep`, not
    `// nosemgrep: <rule-id>` — directory-based `--config` loading
    namespaces rule IDs by path (e.g. `semgrep.rules.<id>`), and that
    prefix depends on how semgrep is invoked, so the qualified form is
    fragile across local/CI/future refactors.
- Rationale:
  - The doc's own bar ("do not enable as blocking CI until each rule has
    positive/negative examples and the current codebase is compliant or
    explicitly exempted") is exactly the condition met here — there was no
    reason to add an unblocking grace period once compliance was already
    verified.
