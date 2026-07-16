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
  - gate 평가 로직은 `internal/gate` 패키지로 캡슐화하며, CLI는 `cmd/slint-gate/main.go`에서만 flag 파싱 및 출력을 담당한다. **(2026-07-09 갱신: `internal/gate`는 이후 post-RC 하드닝 R6에서 `pkg/gate`로 이동했고 `internal/gate`는 더 이상 존재하지 않음 — 이 결정 자체(Go 단일 스택, 로직/CLI 분리)는 여전히 유효하지만 패키지 경로는 `pkg/gate`로 읽을 것. `docs/architecture.md`/`.golangci.yml`은 이미 "formerly internal/gate"로 갱신되어 있었으나 이 항목만 누락됐던 것을 세 번째 `pre-release-adversarial-review`에서 발견.)**
  - `hack/prepare-baseline-update.sh`도 `go run ./cmd/slint-gate` + `jq` 기반으로 재작성되었다.
- Rationale:
  - Go 단일 언어 스택으로 통합하여 Python 런타임/pyyaml 의존을 완전 제거한다.
  - `pkg/gate`(구 `internal/gate`) 단위 테스트로 게이트 로직의 회귀를 방어한다.
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

## D-018: k8sobject's ownerref_missing metric stays same-kind-only, documented as a known limitation

- Date: 2026-07-07
- Status: Accepted
- Decision:
  - `pkg/slo/fetch/k8sobject`'s `*_ownerref_missing_end` metric continues to
    check ownership only within the single Resource kind being listed
    (`Config.Resource`, e.g. "pods"). It does not gain cross-kind owner
    resolution (fetching ReplicaSets/Jobs/etc. to validate a Pod's real
    owner).
  - This is now explicitly documented as a same-kind-only check, in
    `pkg/slo/fetch/k8sobject/list.go`, the package doc comment in
    `fetcher.go`, and a locked-in regression test
    (`TestToEndMetrics_OwnerRefMissing_CrossKindOwnerIsNotResolved`) — a
    normal Pod owned by a ReplicaSet/Job is indistinguishable from one
    whose owner was actually deleted, since the owner's kind is never in
    the listing.
  - The one place this metric was wired into an example gate
    (`pkg/slo/spec/jumi_churn.go`'s `jumi_k8s_ownerref_missing_end`) had its
    Judge rule softened from `LevelFail` to `LevelWarn`, since gating hard
    on a metric with this known false-positive shape for typical Pod
    ownership would be actively bad example code to leave in the tree.
- Rationale:
  - `K8sObjectFetcher`/`ownerref_missing_end` is not part of any default
    spec set (`BaselineV3Specs`/`DefaultSpecs`) — it's only reachable via
    `pkg/slo/spec/jumi_churn.go`, which itself carries `//go:build ignore`
    and isn't compiled into the module. So the "default policy usage does
    not create false positives" bar is already met structurally; the real
    fix needed was making the metric's actual, narrower meaning legible to
    anyone who does wire it up.
  - Implementing full cross-kind owner resolution (listing every plausible
    owner kind — ReplicaSet, Job, StatefulSet, DaemonSet — cross-referencing
    UIDs, and requiring broader RBAC for all of them) is a real architecture
    expansion, not a small fix, and this feature isn't even connected to the
    default `Session`/E2E path yet (see `docs/competition-submission.md`'s
    own roadmap note). Building that out now would be exactly the kind of
    invented, unrequested feature this project has repeatedly declined to
    fabricate under deadline pressure (same judgment as D-016's "no invented
    second profile").

## D-019: container images stay tag-pinned; digest pinning requires an update process first

- Date: 2026-07-07
- Status: Accepted
- Decision:
  - The repo `Dockerfile` (`golang:1.25` builder, `gcr.io/distroless/static:nonroot`
    runtime) and the curlpod default image
    (`pkg/slo/fetch/curlpod/client.go`'s `Image: "curlimages/curl:8.11.0"`)
    stay pinned to specific version tags. None move to digest pinning
    (`@sha256:...`) as part of this decision.
  - This is recorded in `docs/security-model.md` under "Container Image
    Pinning Policy", with an inline comment at each image reference pointing
    back to this entry.
- Rationale:
  - Digest pinning only actually improves reproducibility/supply-chain
    posture if something keeps the digests current (a Renovate/Dependabot
    job or equivalent). This repo has neither today. A manually-pinned
    digest that nobody refreshes is worse than a version tag: it silently
    stops receiving upstream security fixes while looking more "locked
    down" than it is.
  - Version tags (not `:latest`, already specific — `golang:1.25`,
    `curlimages/curl:8.11.0`) give most of the practical reproducibility
    benefit for a CI gate tool while staying human-readable and diffable in
    PRs, which is a real value for a security-relevant file that reviewers
    need to read at a glance.
  - Consumers who need stronger reproducibility guarantees (e.g. air-gapped
    or regulated environments) are expected to pin to digests themselves in
    their own build/registry mirror — that's a per-consumer operational
    decision, not something this repo's examples should force on everyone.
  - If digest pinning is adopted later, it must ship together with the
    automated update process, not as a one-time manual edit — otherwise the
    same rot problem just gets introduced disguised as a hardening step.

## D-020: go.mod's `go` directive lowered from an exact-patch pin to `1.22`

- Date: 2026-07-07
- Status: Accepted
- Decision:
  - `go.mod`'s `go` directive changes from `1.25.5` (the maintainer's
    exact installed toolchain, pinned down to the patch level) to `1.22`
    (major.minor only). Since Go 1.21, this directive is a strictly
    enforced minimum, not a suggestion: a consumer with an older toolchain
    either gets an automatic download (`GOTOOLCHAIN=auto`, the default) or
    an outright build failure (`GOTOOLCHAIN=local`, common in offline/
    security-hardened environments). Pinning the exact patch version the
    maintainer happened to have installed was an unnecessarily strict
    floor for a library meant to be consumed by others.
  - Direct/indirect dependencies that had bumped their own `go` directive
    above 1.22 were downgraded to versions that still declare `go <=1.22`:
    `golang.org/x/net` v0.48.0→v0.35.0, `golang.org/x/sys` v0.39.0→v0.30.0,
    `golang.org/x/tools` v0.39.0→v0.24.0, `golang.org/x/text`
    v0.32.0→v0.22.0, `google.golang.org/protobuf` v1.36.11→v1.36.5. This
    also auto-downgraded `github.com/onsi/ginkgo/v2` v2.22.0→v2.20.2 (a
    test-only dependency) via Go's own MVS resolution.
  - README.md/README(Kor).md/docs/demo.md's stated Go prerequisite updated
    from "Go 1.25+" to "Go 1.22+" to match.
- Verification:
  - The actual language/stdlib feature ceiling of this repo's own code is
    Go 1.20 (`errors.Join` in `pkg/kubeutil/poll.go`) — nothing here
    requires 1.21+ syntax or stdlib additions (no generics-heavy code,
    no `slices`/`maps`/`cmp`, no range-over-func, no `min`/`max` builtins).
  - Both `go 1.20` and `go 1.22` were empirically verified: real
    `golang:1.20` and `golang:1.22` container images (via `podman run`,
    not just simulated with the installed 1.25.5 compiler) were used to
    run `go build ./...`, `go vet ./...`, and `go test ./...` against this
    repo, both passing cleanly.
  - Targeting 1.20 required downgrading `github.com/onsi/gomega` to
    v1.33.0 and `github.com/google/pprof` to a 2021-era pseudo-version —
    a much larger, staler test-tooling diff for only a marginal
    compatibility gain over 1.22, since these are test-only indirect
    dependencies that Go's pruned module graph (1.17+) never loads for a
    downstream consumer who only imports this module's production
    packages. 1.22 was chosen as the better tradeoff: comparable
    consumer-facing compatibility, far smaller and fresher dependency
    diff.
  - Testing at 1.20 also surfaced a genuine, unrelated bug: a classic
    range-loop-variable pointer capture (`Value: &v` inside `for id, v :=
    range values`) in two test fixtures (`pkg/gate/gate_test.go`'s
    `makeMeasurement`, `cmd/slint-gate/baseline_diff_test.go`'s
    `writeDiffSummary`) that Go 1.22's per-iteration loop variable
    semantics had been silently masking. Since the repo settled on 1.22
    (not 1.20) as the floor, no code fix was needed — the bug does not
    manifest at 1.22+, and `golangci-lint`'s `copyloopvar` rule already
    auto-disables below 1.22 and would have flagged a manual `v := v` copy
    as redundant once the directive says 1.22 anyway.
- Rationale:
  - The Dockerfile's `golang:1.25` builder image is intentionally left
    unchanged — a newer toolchain always satisfies a lower `go.mod`
    directive, so the build environment and the consumer-facing minimum
    are independent choices (same reasoning as D-019's image pinning
    policy: don't conflate "what we build with" with "what we require").

## D-021: mandatory pre-release adversarial review; two more flag renames land under the dual-support pattern

- Date: 2026-07-08
- Status: Accepted
- Decision:
  - Before every `git tag`, run the `pre-release-adversarial-review`
    workflow (saved at `.claude/workflows/pre-release-adversarial-review.js`,
    local-only like `CLAUDE.md`/`AGENTS.md`) — 6 independent review
    dimensions (consistency, error-handling, security, test-correctness,
    docs-code alignment, API-naming-consistency) run in parallel, each
    finding is adversarially verified by a separate pass before being
    acted on. This is now a standing rule recorded in `CLAUDE.md`, not a
    one-off exercise.
  - The first run (2026-07-08) found 8 issues, all confirmed real, all
    fixed (see `CHANGELOG.md`'s `[Unreleased]` entry for the full list).
    Two were flag renames following the same dual-support pattern as
    D-016's `promote_to_fail`/`--exit-on` migration:
    - `analyze-dataplane`'s `--fail-on` → `--severity-threshold`
      (`--fail-on` kept as a deprecated alias).
    - The main gate invocation's `--measurement-summary` → `--summary`
      (`--measurement-summary` kept as a deprecated alias), matching what
      every onboarding subcommand already used.
  - Findings that represent a repeatable pattern (not just a one-off bug)
    were also codified as permanent `hack/quality-guardrails.sh` checks —
    e.g. any file shelling out to `kubectl delete` against a
    session/config namespace must reference the shared
    `kubeutil.IsDangerousNamespace` guard, and `SessionConfig`'s
    `StrictnessMode` doc comment must list `RequiredSLIs`. The goal is
    that the next review run (or CI on every push) catches a regression
    of the same shape without needing another full adversarial pass.
- Rationale:
  - Every finding in this review's first run was something CI green had
    already been reporting as "healthy" — none of them were caught by
    `go build`/`go vet`/`go test`/`golangci-lint`/semgrep on their own.
    CI green verifies "it compiles and the tests that exist pass," not
    "the invariants this project claims to guarantee actually hold" — the
    kube-system namespace guard gap is the clearest example: the code
    silently didn't do what `docs/security-model.md` said it
    unconditionally does.
  - Deferring the two flag renames (as originally planned, given the
    2026-07-17 date that was believed to be the full submission deadline)
    was reconsidered once it was clarified that 7/17 is a documentation-only
    milestone and the actual code deadline is mid-August — there was no
    longer a reason to ship a known naming inconsistency instead of fixing
    it under the same safe dual-support pattern already used three times
    in this project (D-016, and now these two).

## D-022: no standalone `pkg/policy` package; the real "public API cleanup" is deduplicating value-map/operator logic

- Date: 2026-07-08
- Status: Accepted
- Decision:
  - The `docs/project-status.yaml`/`CLAUDE.md` backlog item "`pkg/policy`/`pkg/summary` 공개 API 정리" assumed a `pkg/policy` package exists. It does not — `Policy`/`ThresholdRule`/`RegressionCfg`/`ReliabilityCfg` live in `pkg/gate` and have zero external consumers (nothing outside `pkg/gate` constructs a `Policy`; callers only ever go through `gate.Evaluate(gate.Request{...})`). No new `pkg/policy` package was created for this cleanup pass — doing so would mean building public API surface for a consumer that doesn't exist, the same fabrication pattern already rejected in D-016 (no invented second profile) and D-018 (no invented cross-kind owner resolution).
  - What *is* real: three independent reimplementations of "flatten `Summary.Results` into a `map[id]value`" (`pkg/gate`'s `resultValueMap`, `cmd/slint-gate/baseline_diff.go`'s `resultValues`, and `baseline_merge.go` reusing the latter), plus two independent reimplementations of policy operator semantics (`pkg/gate`'s `compareOp`, `cmd/slint-gate/recommend_policy.go`'s `violatesDefault`) and of operator-to-improvement-direction inference (`pkg/gate`'s `lowerIsBetter`/`higherIsBetter`, `baseline_diff.go`'s `lowerIsBetterOperator`/`higherIsBetterOperator`). Each duplicate carried a code comment explicitly justifying it as "not worth expanding pkg/gate's public API for" — reasonable the first time, no longer reasonable at 3 copies of the same logic with real drift risk (a bug fixed in one copy silently doesn't apply to the others).
  - Fix: added `summary.Summary.ResultValues() map[string]float64` to `pkg/slo/summary` (the one package every one of the three call sites already imports) and exported `gate.CompareOp`/`gate.LowerIsBetter`/`gate.HigherIsBetter`. All five duplicate implementations were deleted; call sites now use the shared versions. Also added a package doc comment to `pkg/slo/summary` clarifying that a "baseline" is just a `Summary` (there is no distinct `Baseline` type), since that ambiguity was the other real doc gap the backlog item's own phrasing implied.
- Rationale:
  - `gate.Policy` et al. stay exported from `pkg/gate` (unchanged) rather than being hidden — they're part of `Evaluate`'s already-public contract via `Request`/policy.yaml deserialization internals, and hiding them now would be an unrelated, unrequested breaking change with no consumer asking for it.
  - This also sets up the baseline merge `review-existing` mode (D-023, below): that mode needs the same operator-direction inference `baseline_diff.go` already had, and reusing `gate.LowerIsBetter`/`gate.HigherIsBetter` avoids creating a *third* copy of it instead of consolidating the two that already existed.

## D-023: baseline merge `review-existing` and `force-replace` modes

- Date: 2026-07-08
- Status: Accepted
- Decision:
  - `review-existing`: like `append-new-only` (new SLIs are appended), but for an SLI ID present in both baseline and current summary, the current value replaces the baseline value only when it is a genuine improvement in the direction implied by `policy.yaml`'s threshold operator for that metric (reusing `gate.LowerIsBetter`/`gate.HigherIsBetter`, loaded via `baseline_diff.go`'s existing best-effort `loadMetricDirections`, same pattern already used for diff's improve/weaken wording). A changed value with no recognized direction, or a change that is a regression in the known direction, is left untouched and reported as rejected — identical to `append-new-only`'s behavior for those cases.
  - `force-replace`: current summary's matching-ID values unconditionally overwrite the baseline (plus new SLIs are appended), no direction check. This is an explicit escape hatch for deliberate rebaselining (e.g. after intentionally changing what a metric measures), not a default-safe mode.
  - `baseline_merge_test.go`'s prior assertion that `--mode force-replace` is rejected was deliberately changed to assert against a genuinely bogus mode name instead (it was locking in the mode's prior absence, not a behavior worth preserving).
  - `printMergeReview` now reports an "Existing SLIs updated" section (mode-dependent, shown for `review-existing`/`force-replace` only) distinct from "Existing SLIs unchanged" and "Rejected changes".
  - `runBaselineMerge`'s merge-decision logic was extracted into `computeMergePlan`/`mergeChangeApplies`/`applyMergePlan` — the original single-function version hit `gocyclo`'s complexity-20 threshold (27) once the second mode branch was added.
- Rationale:
  - Both modes stay fully non-interactive (no stdin prompting) — orthogonal to the interactive-wizard work (D-024, below), consistent with every other onboarding command.
  - `review-existing`'s direction-aware auto-update deliberately mirrors the already-established regression-detection direction logic (R2 in the post-RC hardening sprint) rather than inventing new semantics — same metric-direction concept, applied to baseline maintenance instead of gate evaluation.

## D-024: a real interactive wizard, gated on an actual TTY check

- Date: 2026-07-08
- Status: Accepted
- Decision:
  - Added `slint-gate wizard` (`cmd/slint-gate/wizard.go`): an interactive, stdin-prompted walk through the same init → recommend-policy → baseline approve → ci github-actions loop `quickstart` already describes, but confirming with the user and running each step instead of just printing a suggested command.
  - It refuses to run unless stdin is a real terminal, checked via `golang.org/x/term.IsTerminal(int(os.Stdin.Fd()))` — not a bare `fi.Mode()&os.ModeCharDevice != 0` check, which was tried first and is wrong: `/dev/null` is itself a character device, so that check alone reports "interactive" for a piped/CI/`/dev/null`-redirected invocation and the wizard would then hang forever on its first prompt, exactly the failure mode this command was originally deferred to avoid. `golang.org/x/term` was added as a new direct dependency for this (an official, minimal `golang.org/x` package, same trust tier as the `x/net`/`x/sys`/`x/tools` already depended on) — picked at `v0.29.0` specifically (not `@latest`) since a bare `go get ... @latest` re-bumped `go.mod`'s `go` directive back up past `1.22` and `golang.org/x/sys` back up, undoing D-020; pinning the version kept both unchanged.
  - State detection is shared with `quickstart` via a new `cmd/slint-gate/onboarding_state.go` (`inspectOnboardingState`/`nextOnboardingStep`), extracted out of `quickstart.go` so the two commands can never disagree about what state the project is in. `quickstart.go`'s `nextCommand` now formats the same `onboardingStep` enum into a string rather than duplicating the precedence logic.
  - Each confirmed step calls the corresponding subcommand's own unexported `run*(args []string) error` directly (`runInit`, `runRecommendPolicy`, `runBaselineApprove`, `runInspect`, `runCIGithubActions`) with programmatically-built args — there is exactly one implementation of each step's behavior, not a wizard-specific reimplementation.
  - `runWizardStep`'s per-step logic was split into one function per step (`wizardStepInit`, `wizardStepRecommendPolicy`, etc.) to stay under `gocyclo`'s complexity threshold, same pattern as D-023's `baseline_merge.go` extraction.
- Rationale:
  - This directly answers the concern the original Sprint 6 deferral (see "Quickstart Status" in `docs/sli-gate-onboarding-ux.md`) left open without a response: a stdin-prompting wizard is safe to build once it can reliably tell a real terminal apart from CI/piped stdin and hard-refuse the latter instead of hanging.
  - `quickstart` is unchanged in behavior and remains the correct choice for CI/scripted use; `wizard` is strictly additive, not a replacement.

## D-025: PodSpec/JSON injection via unescaped `serviceAccountName`/`Image` in curlpod's `--overrides` payload

- Date: 2026-07-08
- Status: Accepted
- Decision:
  - `pkg/slo/fetch/curlpod/client.go`'s `RunOnce` built the `kubectl run --overrides` JSON payload with a raw `fmt.Sprintf` template. `podName`/`ns` were already validated (DNS-1123 label via `ValidateMetricsURL`), but `serviceAccountName` and `c.Image` were spliced in unescaped. A crafted `serviceAccountName` (e.g. containing `","hostNetwork":true,...`) could inject sibling PodSpec fields — `hostNetwork`, a `hostPath` volume, a privileged `initContainer` — directly contradicting `docs/security-model.md`'s "Privileged pod and hostPath are rejected in generated/default resources" invariant. Found by the second `pre-release-adversarial-review` run (2026-07-08) and independently reproduced (a standalone PoC test confirmed the crafted value produces valid JSON with `hostNetwork:true` and a `hostPath` volume mounting `/`).
  - Fixed two ways: (1) `serviceAccountName` is now validated as a DNS-1123 label (`isValidDNSLabel`, the same check `ns`/`metricsSvcName` already get) before it's used at all — rejects the exploit shape outright with a clear error, since valid Kubernetes ServiceAccount names can never contain the characters needed to break out of a JSON string. (2) The entire `--overrides` payload is now built by `encoding/json.Marshal` of a typed `podOverride` struct (new in `client.go`) instead of `fmt.Sprintf` string interpolation — this closes the injection class structurally, for every field, not just the two known-unsafe ones, and matches the shape the labels JSON already used (`json.Marshal(podLabels)`) but the rest of the payload didn't.
  - Added `.semgrep/rules/kube-slint-no-raw-json-splice-in-podspec.{yaml,go}`: flags any `fmt.Sprintf` whose format string contains a `"key":"%s"`/`%q`-shaped JSON key literal, so this exact pattern can't silently reappear elsewhere. Verified `semgrep --test .semgrep/rules` passes and `make semgrep` finds 0 findings against the real tree.
  - Updated `hack/quality-guardrails.sh`'s curlpod securityContext checks, which grepped for the old literal JSON text (`"allowPrivilegeEscalation": false`, etc.) and would have false-failed against the new Go struct-literal form; also added a new check asserting the `isValidDNSLabel(serviceAccountName)` guard exists.
  - Added regression tests: `TestRunOnce_RejectsServiceAccountNamePodSpecInjection` (end-to-end, the exact PoC payload) and `TestPodOverrideMarshal_ServiceAccountNameCannotBreakOutOfJSON` (verifies the marshal-based structural fix independent of the validation guard).
- Rationale:
  - Both layers (validate + marshal-don't-splice) are kept rather than picking one: validation gives a clear, early, human-readable error for the common case; the typed-struct marshal is the actual structural fix and protects any future field added to the payload without needing its own bespoke validation.
  - `Image` was not given DNS-label validation (image references have a different, more permissive grammar) — the marshal fix alone closes its injection vector, and inventing an image-reference validator was judged out of scope for this fix (the vulnerability was the injection, not "is this a syntactically valid image reference").

## D-026: `captureStdout` test helper now drains concurrently, not after `fn()` returns

- Date: 2026-07-08
- Status: Accepted
- Decision:
  - `cmd/slint-gate/inspect_test.go`'s `captureStdout` (shared across 8 test files) redirected `os.Stdout` to an `os.Pipe()`, ran `fn()` to completion, and only then started draining the read end. `os.Pipe()`'s write end has a bounded kernel buffer (commonly 64KiB on Linux, not a portable guarantee) — any `fn()` writing more than that in one call would block on `write(2)` forever, since nothing was reading yet. Found by the second `pre-release-adversarial-review` pass; independently confirmed by reproducing the old implementation's deadlock in a standalone throwaway test (killed by `go test`'s own timeout after >1 minute).
  - Fixed by starting a goroutine that drains the pipe into a `bytes.Buffer` via `io.Copy` *before* `fn()` runs, synchronized by a `done` channel closed when the drain completes; `captureStdout` waits on that channel after closing the write end before returning the buffer's contents.
  - Added `TestCaptureStdout_DoesNotDeadlockOnOutputLargerThanPipeBuffer`, which writes 512KiB inside `fn()` and asserts completion within a bounded timeout via `select`.
- Rationale:
  - This doesn't fire today (no current CLI output exceeds 64KiB in one `captureStdout` call), but it's exactly the kind of incidental/version-dependent-behavior bug this project's test-correctness review dimension targets — a future CLI feature with larger output would otherwise hang the entire `go test` run with no useful diagnostic.

## D-027: a failed curl-pod scrape is no longer indistinguishable from a successful empty one

- Date: 2026-07-09
- Status: Accepted
- Decision:
  - `pkg/slo/fetch/curlpod/client.go`'s `WaitDone` (via the former `isTerminal`) treated pod phase `Succeeded` and `Failed` identically — both just meant "stop polling." `CurlPod.Run` then unconditionally fetched and returned the pod's logs regardless of which terminal phase it reached. Since the in-pod script uses `curl --fail-with-body`, a non-2xx response (e.g. an RBAC 403, wrong port, or a proxied error page) makes the pod phase `Failed` while still writing the response body to stdout — and `promtext.ParseTextToMap` silently `continue`s past any line it can't parse as a metric (e.g. a JSON `Status` error body), yielding an empty-but-successful `map[]`/`err=nil` sample. `pkg/slo/engine` only sets `Reliability.CollectionStatus=Failed` when `Fetch()` itself returns a Go error, which this path never produced — so a genuinely failed scrape reported `CollectionStatus=Complete` with the affected SLIs quietly `StatusSkip`, indistinguishable from an operator legitimately exposing zero matching metrics. This directly contradicted `docs/architecture.md`'s documented reliability contract. Found by the third `pre-release-adversarial-review` pass (2026-07-09) and independently re-confirmed against the actual code before fixing.
  - Fixed by having `WaitDone` distinguish the two terminal phases: `Succeeded` still returns `nil`, but `Failed` now returns a new exported `ErrPodFailed` sentinel (wrapped with the pod's namespace/name via `%w`). `CurlPod.Run` treats an `ErrPodFailed` result specially — it still best-effort fetches the pod's logs (since that's the actual diagnostic value of `--fail-with-body`, e.g. the 403 body), but returns them embedded in the error (redacted via `evidence.RedactString`, the same redaction contract already applied to every other error/log surface in this codebase) instead of as a successful sample. A non-phase error (kubectl call itself failing, context deadline) skips the log fetch entirely, since the pod never reached a terminal phase.
  - This required no changes to `pkg/slint/fetcher_curlpod.go` or `pkg/slo/engine` — both already correctly propagate a non-nil `Fetch()`/`Run()` error into `CollectionStatus=Failed`; the bug was entirely that no such error was ever being produced for this specific failure mode.
  - Added regression tests at both layers: `client_test.go`'s `TestWaitDone_Failed_ReturnsErrPodFailed`/`TestWaitDone_Succeeded_ReturnsNil`/`TestWaitDone_KubectlError_PropagatesWithoutErrPodFailed`, and a new `curlpod_test.go` covering `CurlPod.Run`'s full lifecycle (`TestCurlPodRun_Failed_ReturnsErrorNotLogs`, `TestCurlPodRun_Failed_ErrorIncludesRedactedPodOutput`, `TestCurlPodRun_Failed_StillCleansUpPod`, `TestCurlPodRun_KubectlWaitError_DoesNotFetchLogs`).
- Rationale:
  - This is the project's core value proposition (a shift-left SLI *guardrail* whose entire point is that a broken measurement must never silently look like a clean one), so it was fixed directly rather than deferred or merely pattern-guarded — there's no grep/semgrep pattern that generalizes "does this specific failure path get correctly classified," it needed real logic.
  - Redacting the embedded pod output (rather than passing it through raw) matters because this error can end up in `Reliability.BlockedReason`, a field persisted into `sli-summary.json` — the same artifact class `evidence.RedactString` already protects elsewhere (`pkg/kubeutil/runner.go`'s command-error path).

## D-028: five more copies of the captureStdout pipe-buffer deadlock, consolidated

- Date: 2026-07-09
- Status: Accepted
- Decision:
  - The third `pre-release-adversarial-review` pass found the same defect class as D-026 (`cmd/slint-gate/inspect_test.go`'s `captureStdout`, fixed 2026-07-08) duplicated five more times: `pkg/gate/gate_test.go`'s `captureStderr`, `cmd/slint-gate/diagnose_test.go`'s `capturePrintDiagnostics`, and three inline `os.Pipe()` capture blocks in `cmd/slint-gate/init_test.go` (`TestRunInit_WithServiceOverride`, `TestRunInit_DefaultPlaceholders`, and a `capture` closure in `TestPrintDiscoveryResult_DiscoverErrorIsDistinctFromNoCandidates`). All five drained their pipe only after the captured function returned, the same latent deadlock-on-output-larger-than-the-OS-pipe-buffer risk.
  - `pkg/gate/gate_test.go`'s `captureStderr` is in a different package from the already-fixed `captureStdout`, so it got its own independent fix using the identical concurrent-drain-goroutine pattern.
  - The three `cmd/slint-gate` sites are in the *same* package (`main`) as the already-fixed `captureStdout` — rather than fixing each site independently (which would just be a sixth/seventh/eighth copy of the same fix), they were consolidated to call `captureStdout` directly. `diagnose_test.go`'s `capturePrintDiagnostics` became a one-line wrapper (`captureStdout(t, func() { printDiagnostics(result) })`) instead of duplicating pipe-capture logic; the two `init_test.go` inline blocks were replaced with direct `captureStdout` calls.
  - Added `check_test_capture_helper_consolidation` to `hack/quality-guardrails.sh`: fails if `os.Pipe()` appears in any `cmd/slint-gate`/`pkg/gate` test file other than the two canonical fixed helpers, so a future ad-hoc capture helper can't reintroduce this pattern a sixth time.
- Rationale:
  - Consolidating onto the existing fixed helper (rather than re-fixing each site independently) is strictly better here: one implementation to maintain, and it matches this project's general anti-duplication stance (see D-022's `ResultValues`/`CompareOp` consolidation) — three near-identical bug fixes in one PR is a stronger signal to standardize than to keep patching copies.

## D-029: Real-usage SLI governance hardening sprint is accepted as the next development track

- Date: 2026-07-16
- Status: Accepted (Sprint A in progress)
- Decision:
  - A new real-usage hardening sprint is accepted to address four issues found
    while using kube-slint from a consumer/development-agent workflow:
    1. kube-slint can compute SLIs that are never covered by policy or test
       assertions, and currently does not surface that as a first-class
       governance risk.
    2. Window/range/latency SLIs such as startup latency, request latency
       percentiles, and burn-rate style checks do not fit the current
       two-point engine without a design extension.
    3. Public examples over-emphasize raw Prometheus key construction
       (`UnsafePromKey`) even though kube-slint's product identity is
       source-neutral operational SLI guardrails.
    4. Non-Prometheus source adapters are possible through
       `MetricsFetcher`/`SnapshotFetcher`, but common HTTP JSON/expvar
       adapter boilerplate is not yet provided or documented as a first-class
       path.
  - Sprint A, "Guardrail Coverage & Source-Neutral UX", is the immediate
    implementation sprint. It may add policy-coverage diagnostics and
    source-neutral wording/API examples, but it must not claim that
    window-based SLI semantics have shipped.
  - Sprint B, "Non-Prometheus Source Adapters & Window SLI Design", follows
    Sprint A. It may add small source-adapter examples or packages, but the
    window/range engine extension must start as a design decision before
    runtime behavior changes.
  - The sprint schedule and acceptance criteria live in
    `docs/real-usage-sli-governance-sprint.md`.
- Rationale:
  - D-001 defines kube-slint as a shift-left operational SLI guardrail, not
    a Prometheus-specific metrics tool. The current implementation is already
    source-extensible at the fetcher interface boundary, but its examples and
    ergonomics make Prometheus text scraping feel like the only intended
    path.
  - D-002/D-008 keep measurement, policy evaluation, and CI failure separate.
    A coverage diagnostic preserves that separation while making "measured
    but not gated" visible instead of silently relying on humans to notice.
  - `docs/verification-sources.md` already records the two-point engine
    boundary and the unimplemented `WindowFetcher` direction. Treating
    latency/window SLIs as a design-first track avoids fragmenting the clean
    two-point path with ad hoc semantics.

## D-030: Non-Prometheus JSON endpoints are Tier 1 sources; window SLIs remain design-first

- Date: 2026-07-16
- Status: Accepted (Sprint B in progress)
- Decision:
  - `pkg/slo/fetch/jsonendpoint` is accepted as a small Tier 1 source adapter
    for HTTP JSON/expvar-style endpoints. It implements
    `fetch.SnapshotFetcher`, flattens numeric JSON leaves into dot-separated
    input keys, ignores non-numeric leaves, and reuses the existing two-point
    engine without changing `SLISpec`, summary, or gate semantics.
  - JSON/expvar support is deliberately limited to scalar numeric leaves. It
    does not attempt to interpret histograms, distributions, timestamps, or
    event streams.
  - Window/range/latency semantics remain a design-first track in
    `docs/window-sli-design.md`; no runtime window engine behavior ships as
    part of this decision.
- Rationale:
  - D-029 identified that kube-slint's product identity is source-neutral even
    though its examples and default fetchers made Prometheus text scraping feel
    like the only intended path. A small JSON endpoint fetcher proves the
    source-neutral boundary without inventing a new engine model.
  - expvar and simple status JSON endpoints naturally produce keyed numeric
    scalars, so they satisfy `docs/verification-sources.md`'s Tier 1 rule:
    their result can be expressed as `map[string]float64` at a single point in
    time.
  - Latency percentiles, burn-rate, and soak analysis do not satisfy that rule.
    Keeping them behind a separate design avoids overloading the two-point
    path with misleading semantics.

## D-031: Window SLI engine foundation supports scalar window aggregation

- Date: 2026-07-16
- Status: Accepted (initial implementation)
- Decision:
  - `fetch.WindowFetcher` is now a real optional interface:
    `FetchRange(ctx, start, end) ([]fetch.Sample, error)`.
  - `engine.ExecuteRequest` accepts an optional `WindowFetcher`. The existing
    two-point `MetricsFetcher` path remains unchanged for `delta`/`start`/`end`
    specs, and specs with no window compute mode do not require a window
    fetcher.
  - The first implemented window compute modes are scalar aggregations over
    numeric sample values: `window_min`, `window_max`, `window_avg`,
    `window_p95`, and `window_p99`.
  - Window fetcher absence or failure does not become a correctness-test
    failure. Missing window support skips the affected SLI and makes evaluation
    partial; a window fetch error records failed collection while still
    returning a summary.
  - This does not implement histogram quantile semantics, burn-rate/ratio
    semantics, PromQL range query fetchers, or SessionConfig-level window
    wiring.
- Rationale:
  - D-029 identified latency/window SLIs as a real consumer gap, and D-030
    deliberately deferred runtime semantics until the boundary was explicit.
    The scalar aggregation subset is the smallest useful engine extension that
    follows that boundary.
  - Keeping `WindowFetcher` separate from `MetricsFetcher` avoids pretending
    that p95/p99 can be computed from two point samples.
  - Summary and gate schemas do not need to change for this initial subset:
    a window SLI still emits one scalar `SLIResult.Value`, so existing
    threshold and regression evaluation can operate unchanged.

## D-032: Session-level window wiring, Prometheus range source, ratio mode, and policy coverage governance

- Date: 2026-07-16
- Status: Accepted (implemented)
- Decision:
  - `pkg/slint.SessionConfig` now accepts an optional `WindowFetcher`, and
    `Session.End()` passes it through to the engine. Window-only specs do not
    create the default curlpod point fetcher.
  - `pkg/slo/fetch/promrange` is accepted as a concrete `WindowFetcher` for
    Prometheus `/api/v1/query_range`. It parses matrix results into
    `[]fetch.Sample` and derives sample keys from Prometheus metric labels.
  - `window_ratio` is accepted as the first ratio/burn-rate-style scalar
    window mode. It computes `sum(input[0]) / sum(input[1])` over the returned
    window samples and skips on missing inputs or zero denominator.
  - `policy.coverage` is added to `slint.policy.v1` as an opt-in governance
    check:
    - `coverage.required: true` reports measured SLIs that have no threshold
      rule and are not listed as informational.
    - `coverage.informational: [...]` explicitly marks measured SLIs that are
      intentionally not gated.
    - `promote_to_fail: ["coverage_gap"]` can promote a coverage gap from WARN
      to FAIL. Without that promotion, coverage gaps are WARN.
- Rationale:
  - D-031 made window evaluation possible at the engine boundary. Session-level
    wiring is required for normal consumers to use it without bypassing
    `pkg/slint`.
  - PromQL range queries are a concrete source for latency/range values, but
    they still enter through the source-neutral `WindowFetcher` boundary rather
    than becoming a special engine dependency.
  - `window_ratio` covers a useful class of error-rate/burn-rate checks while
    keeping the first ratio semantics explicit and scalar.
  - Coverage governance closes the remaining "measured but not gated" gap
    without making advisory diagnostics fail CI by default.
