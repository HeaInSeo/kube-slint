# Changelog

모든 변경사항은 이 파일에 기록됩니다.
형식은 [Keep a Changelog](https://keepachangelog.com/ko/1.1.0/)를 따릅니다.

## [Unreleased]

### Added

- `slint-gate inspect --policy` now prints concrete next actions for
  measured-but-not-policy-covered SLIs: add a threshold, mark informational, or
  remove/ignore accidental signals.
- README/README(Kor) now include a source-selection guide for default curlpod,
  portforward, jsonendpoint, and promrange.
- Added `docs/window-sli-example.md` with a complete `promrange`,
  `window_p95`, `window_ratio`, and policy example.
- `slint-gate recommend-policy` now emits a strict `coverage` block with
  profile informational SLIs prefilled under `coverage.informational`.
- Coverage governance is strict by default in generated policies:
  `coverage.required: true` and `coverage_gap` in `promote_to_fail`. Omitted or
  empty promotion lists now include `coverage_gap` in the default promotion set.
- README, onboarding, and verification-source docs now use consistent
  point/snapshot/range/window source terminology while preserving the public Go
  API names.
- Removed stale coverage-governance wording from diagnostics and status docs
  after D-034 made coverage gaps strict by default.
- `slint-gate inspect --policy`: best-effort advisory policy coverage
  diagnostics. The command now reports measured SLIs that are not covered by
  threshold/regression policy and policy-covered SLIs missing from the current
  summary. This is informational only; it does not change gate results or CI
  failure behavior.
- `pkg/slo/fetch/jsonendpoint`: a source-neutral HTTP JSON/expvar
  `SnapshotFetcher` for endpoints that expose numeric JSON leaves. Keys are
  flattened with dot separators (for example, `memstats.Alloc`) and can be
  referenced from `SLISpec.Inputs` like any other fetch sample key.
- Initial scalar window SLI engine foundation: `fetch.WindowFetcher`,
  `ExecuteRequest.WindowFetcher`, and compute modes `window_min`,
  `window_max`, `window_avg`, `window_p95`, and `window_p99`.
- Session-level window wiring via `SessionConfig.WindowFetcher`, a concrete
  Prometheus range source (`pkg/slo/fetch/promrange`), `window_ratio`, and
  opt-in policy coverage governance (`coverage.required`,
  `coverage.informational`, and `promote_to_fail: ["coverage_gap"]`).

### Documented

- Accepted the Source And Window UX Sprint (D-033), covering inspect
  next-action wording, source selection docs, a promrange window example,
  coverage policy recommendation flow, and source-neutral terminology. No
  runtime behavior changed in this planning update.
- Accepted the real-usage SLI governance hardening sprint (D-029), with Sprint
  A focused on measured-but-not-gated SLI diagnostics and source-neutral UX,
  and Sprint B focused on non-Prometheus adapter ergonomics plus window/range
  SLI design. README wording now presents `MetricsFetcher`/`SnapshotFetcher`
  as the source-neutral input boundary and Prometheus helpers as conveniences,
  not engine requirements.
- Added `docs/window-sli-design.md` and D-030 to keep latency/window/range SLI
  support design-first. D-031 records the initial scalar window aggregation
  implementation and its explicit non-goals.
- D-032 records Session-level window wiring, Prometheus range fetching,
  `window_ratio`, and opt-in policy coverage governance.

## [1.6.0] - 2026-07-09

### Added

- `slint-gate wizard`: a real interactive, stdin-prompted onboarding flow (init → recommend-policy → baseline approve → ci github-actions), previously deferred in Sprint 6 pending a TTY/CI-safety answer. Refuses to run unless stdin is a real terminal (`golang.org/x/term.IsTerminal`), so CI/piped invocations never hang on a prompt. Shares state detection with `quickstart` (`cmd/slint-gate/onboarding_state.go`) and calls each subcommand's own function directly rather than reimplementing it. See D-024.
- `slint-gate baseline merge --mode review-existing`: updates an existing baseline SLI's value only when the current measurement is a confirmed improvement in the direction implied by `policy.yaml`'s threshold operator for that metric. A change with no recognized direction, or a regression, is still rejected and left unchanged.
- `slint-gate baseline merge --mode force-replace`: unconditionally overwrites existing baseline values regardless of direction — an explicit escape hatch for deliberate rebaselining, not the default. `append-new-only` remains the default mode. See D-023.

### Fixed

- **F4**: `pkg/slo/fetch/promtext`'s exposition-format parser could not handle a label value containing whitespace (e.g. `metric{path="/foo bar"} 1`) — a naive `strings.Fields` split broke the label block apart, handing a truncated fragment to `strconv.ParseFloat` and erroring out the entire scrape over one line. Replaced with a parser that locates the label block's matching unquoted `}` instead of whitespace-splitting the whole line.

A second `pre-release-adversarial-review` run against this batch (see
`CLAUDE.md`'s standing pre-tag rule) found 8 more issues, all confirmed
real, all fixed (none deferred):

- **Security (high)**: `pkg/slo/fetch/curlpod/client.go`'s `RunOnce` spliced `serviceAccountName`/`Image` unescaped into a hand-built `--overrides` JSON payload via `fmt.Sprintf` — a crafted `serviceAccountName` could inject sibling PodSpec fields (`hostNetwork`, a `hostPath` volume, a privileged container), defeating the documented "never privileged / never hostPath" invariant. Independently reproduced. Fixed by validating `serviceAccountName` as a DNS-1123 label and rebuilding the entire payload via `encoding/json.Marshal` of a typed struct instead of string interpolation. See D-025.
- `cmd/slint-gate/main.go`'s `inspect` and `quickstart` dispatch cases exited silently on error, unlike every sibling case (`init`/`recommend-policy`/`wizard`/`ci github-actions`), which print `slint-gate <cmd>: %v` first. Verified live: `slint-gate inspect --profile does-not-exist` printed nothing.
- `README.md`/`README(Kor).md`'s flag table self-referentially called `--summary` its own deprecated alias; the real deprecated alias is `--measurement-summary`.
- `docs/competition-submission.md` cited a stale "latest release: v1.5.2" and its subcommand table omitted the shipped `wizard` command.
- `.github/actions/slint-gate/action.yml` never gained a `summary` input alias matching the CLI's `--summary`/`--measurement-summary` rename, unlike the fully-symmetric `exit-on`/`fail-on` treatment elsewhere in the same file.
- `cmd/slint-gate/inspect_test.go`'s shared `captureStdout` test helper (used across 8 test files) drained its pipe only after `fn()` returned — any CLI output larger than the OS pipe buffer (~64KiB on Linux) would deadlock the test. Independently reproduced the deadlock in a standalone repro before fixing. See D-026.

Permanent guardrails added for the repeatable patterns: `.semgrep/rules/kube-slint-no-raw-json-splice-in-podspec`, and two new `hack/quality-guardrails.sh` checks (`check_cli_dispatch_error_printing`, `check_flag_deprecation_docs`).

A third `pre-release-adversarial-review` run found 9 more findings (2 were
duplicate reports of the same bug — 8 unique), all confirmed real, all
fixed (none deferred):

- **Reliability (high)**: `pkg/slo/fetch/curlpod`'s `WaitDone` treated pod phase `Succeeded` and `Failed` identically, so `CurlPod.Run` always returned the pod's logs regardless of outcome. Since the in-pod script uses `curl --fail-with-body`, a non-2xx response (RBAC 403, wrong port, etc.) set phase `Failed` while still writing the response body to stdout — and the promtext parser silently skips lines it can't parse as metrics, so a JSON error body parsed to an empty-but-successful `map[]`/`nil`. The engine only sets `CollectionStatus=Failed` on a Go error from `Fetch()`, which this path never produced, so a genuinely failed scrape reported `CollectionStatus=Complete` with the affected SLIs quietly skipped — indistinguishable from an operator legitimately exposing zero metrics. Contradicted `docs/architecture.md`'s documented reliability contract. `WaitDone` now returns a new `ErrPodFailed` sentinel for phase `Failed`; `CurlPod.Run` still best-effort fetches logs for diagnostics but returns them embedded (redacted) in the error instead of as a successful sample. See D-027.
- `slint-gate ci github-actions` generated the deprecated `measurement-summary:` action input in its printed snippet instead of the preferred `summary:`, even though every other onboarding command and `action.yml` itself had already migrated — the one command whose entire job is handing a new user a working, best-practice snippet handed them the deprecated key.
- Two label-selector-based `kubectl delete` cleanup paths used singular `pod`, and one name-based delete used plural `pods` — standardized (name-based = singular, selector-based = plural, matching this codebase's existing `kubectl get` convention).
- Five more copies of the `captureStdout`/`captureStderr` pipe-buffer deadlock (see the previous round's D-026): `pkg/gate/gate_test.go`'s `captureStderr`, `cmd/slint-gate/diagnose_test.go`'s `capturePrintDiagnostics`, and three inline capture blocks in `init_test.go`. `captureStderr` got its own fix; the three `cmd/slint-gate` sites were consolidated onto the already-fixed `captureStdout`. See D-028.
- `docs/DECISIONS.md`'s D-012 still described gate logic as living in `internal/gate`, renamed to `pkg/gate` in the R6 post-RC hardening move; `pkg/slint/attach.go`'s commented-out pseudo-code `Config` type no longer matched real `SessionConfig` (wrong type name, a phantom field, missing most real fields) — removed in favor of a doc comment pointing at the real definition.

Permanent guardrails added: `check_kubectl_delete_pod_resource_naming` and `check_test_capture_helper_consolidation` in `hack/quality-guardrails.sh`, plus an extension to `check_flag_deprecation_docs` covering the `ci github-actions` generator.

### Changed

- `pkg/gate/gate.go` (866 lines) split into `types.go`, `policy.go`, `measurement.go`, `threshold.go`, `regression.go`, `reliability.go`, and a slimmed-down orchestration-only `gate.go`. No behavior change.
- Deduplicated three independent "flatten a Summary into a `map[id]value`" implementations into `summary.Summary.ResultValues()`, and two independent copies each of policy-operator comparison and improvement-direction inference into exported `gate.CompareOp`/`gate.LowerIsBetter`/`gate.HigherIsBetter`. `pkg/policy` was not created as a standalone package — `gate.Policy` et al. have zero external consumers, so that would have been building public API for a consumer that doesn't exist. See D-022.

## [1.5.3] - 2026-07-08

### Fixed

Findings from a `pre-release-adversarial-review` workflow run (6 review
dimensions, adversarial-verified): 8 raw findings, 8 confirmed real, all
fixed (none deferred). See `.claude/workflows/pre-release-adversarial-review.js`
and `hack/quality-guardrails.sh`'s "public API doc-comment sync" / "rbac
guardrails" sections for the permanent regression checks added alongside
these fixes.

- **`baseline merge --output` had no overwrite guard**: unlike `init`/
  `recommend-policy`/`baseline approve`, `baseline merge`'s `--output` had
  no `--force` flag — an explicit `--output` pointing at a pre-existing,
  unrelated file was silently clobbered. Added `--force`, guarding only
  the case where `--output` differs from `--baseline` (in-place merge, the
  default, is still allowed to overwrite the baseline it just read from —
  that's the point of merge).
- **`Session.Cleanup()`/`SweepOrphansWithResult()` didn't enforce the
  kube-system namespace guard**: `docs/security-model.md` documents
  kube-system/kube-public/kube-node-lease rejection as an unconditional
  default, but that check only ever lived in `curlpod.Client.RunOnce`
  (pre-scrape). A misconfigured `Namespace: kube-system` sailed straight
  through to a real `kubectl get/delete pods`. The check
  (`kubeutil.IsDangerousNamespace`, newly extracted to `pkg/kubeutil` so
  `curlpod` and `pkg/slint` share one definition) is now enforced on both
  paths, gated by the same `DangerouslyAllowKubeSystemNamespace` opt-in.
- **`IsCertManagerCRDsInstalled`/`IsPrometheusOperatorCRDsInstalled`
  silently converted any `kubectl` failure into `false`**: a connectivity/
  permission error was indistinguishable from "CRDs genuinely not
  installed." Both now log the real underlying error.
- **`loadPolicy`/`loadMeasurement` discarded non-NotExist `os.ReadFile`
  errors**: a permission or filesystem error was mapped to the same
  generic invalid/corrupt state as an actual YAML/JSON syntax error,
  misdirecting `diagnose.go`'s hints. The real error is now surfaced via
  `PolicyWarnings` (policy) or stderr (measurement).
- **README/README(Kor).md described `InsideAnnotation` as a working
  "precise semantic-boundary" measurement mode**: it's explicitly
  reserved/unimplemented in code (behaves identically to
  `InsideSnapshot`). Both READMEs now say so.
- **`SessionConfig.StrictnessMode`'s doc comment omitted `RequiredSLIs`**:
  `propagation.go` implements a fourth, distinct mode not mentioned in the
  public field's comment. Fixed.

### Changed

- **`analyze-dataplane`'s `--fail-on` renamed to `--severity-threshold`**:
  it collided in name (not meaning) with the gate command's deprecated
  `--fail-on`/`--exit-on` pair — same flag name, unrelated value domain
  (finding-severity vs. gate-result). `--fail-on` keeps working as a
  deprecated alias on this subcommand.
- **The main `slint-gate` gate invocation's `--measurement-summary`
  renamed to `--summary`**: every onboarding subcommand (`inspect`,
  `recommend-policy`, `baseline diff/approve/merge`, `ci github-actions`)
  already used `--summary` for the identical artifact.
  `--measurement-summary` keeps working as a deprecated alias.
  `.github/actions/slint-gate`'s internal CLI invocation switched to
  `--summary` to avoid the new deprecation warning on every run (the
  action's own `measurement-summary:` input name is unchanged — that's a
  separate, stable YAML contract).
- Lowered `go.mod`'s `go` directive from an exact-patch pin (`1.25.5`) to
  `1.22`. Since Go 1.21 this directive is a strictly enforced minimum, so
  pinning the maintainer's exact installed toolchain was an unnecessarily
  strict floor for consumers. Verified in real `golang:1.20` and
  `golang:1.22` containers (not just the installed 1.25.5 compiler) that
  both build/vet/test cleanly; 1.22 was chosen over 1.20 since it needs a
  far smaller, fresher dependency downgrade (only `golang.org/x/net`,
  `x/sys`, `x/tools`, `x/text`, `google.golang.org/protobuf`) for
  comparable consumer-facing compatibility. README.md/README(Kor).md/
  docs/demo.md's stated Go prerequisite updated to match. See
  `docs/DECISIONS.md` D-020.

## [1.5.2] - 2026-07-07

### Documented

- **`pkg/slo/fetch/k8sobject`'s `ownerref_missing` metric**: documented as a
  same-kind-only check (a Pod's usual ReplicaSet/Job owner is a different
  kind and is never resolved, so a healthy Pod looks identical to one whose
  owner was actually deleted) instead of silently implying general
  owner-deletion detection. Locked in with a regression test
  (`TestToEndMetrics_OwnerRefMissing_CrossKindOwnerIsNotResolved`). The one
  example gate using this metric (`pkg/slo/spec/jumi_churn.go`) had its
  Judge rule softened from `LevelFail` to `LevelWarn` accordingly. See
  `docs/DECISIONS.md` D-018.
- **Container image pinning policy**: recorded that the Dockerfile and
  curlpod default image stay tag-pinned (not digest-pinned) until an
  automated digest-refresh process exists, with inline comments at each
  image reference. See `docs/security-model.md` "Container Image Pinning
  Policy" and `docs/DECISIONS.md` D-019.

### Changed

- `README(Kor).md` switched from polite/formal endings (-습니다/-하세요) to
  a plain declarative tone (-다/-함) throughout, matching a dry, expository
  technical-doc style. Code blocks and literal config values are
  untouched.

## [1.5.1] - 2026-07-07

### Fixed

- **`.github/actions/slint-gate` composite action**: the CLI is now always invoked
  with `--exit-on NEVER` in the first step; the pass/fail decision (per the
  action's `exit-on`/`fail-on` input) is made only in the final "Check gate
  result" step, after the summary artifact and step outputs are already
  captured. Previously, a default-configured run (`exit-on` unset, `fail-on`
  defaulting to `FAIL_OR_NOGRADE`) would have the CLI itself exit 1 on a real
  FAIL/NO_GRADE result, which — under `set -euo pipefail` — aborted the rest
  of that step and skipped the artifact-upload and result-check steps
  entirely (no `if: always()` guard), so a policy failure produced no
  uploaded artifact and no `gate-result`/`evaluation-status` outputs.
- **`slint-gate init`**: added `--force` to guard `--output` (`policy.yaml`)
  and `--emit-rbac` against silent overwrite, matching the existing
  `recommend-policy --force` / `baseline approve --force` convention. Previously
  `init` was the only onboarding command that unconditionally overwrote an
  existing file.
- **`slint-gate init` service discovery**: a failed `kubectl get svc` call
  (not on PATH, no cluster access, timeout) is no longer swallowed into a
  silent empty candidate list indistinguishable from "ran fine, found
  nothing." `discoverMetricsServices` now returns the error, and `init`
  prints why discovery failed instead of just "no metrics services
  auto-detected."

### Changed

- Internal (non-public-API) naming in `pkg/gate` and `cmd/slint-gate` no
  longer centers on the deprecated `fail_on`/`--fail-on` vocabulary now that
  the public names are `promote_to_fail`/`--exit-on`: `makeFailOn` →
  `makePromotionSet`, `allowedPolicyFailOn` → `allowedPromotionValues`,
  `normalizeFailOn` → `normalizePromotionValue`, `isValidFailOn` →
  `isValidExitOn`, `shouldFailOn` → `shouldExitOn`. No behavior change; the
  `analyze-dataplane` subcommand's own unrelated `--fail-on` flag (which was
  never renamed to `--exit-on` and isn't part of this migration) is
  untouched.
- Unified code comments and `slint-gate`'s runtime diagnostic messages
  (`cmd/slint-gate/diagnose.go`'s `MEASUREMENT_INPUT_MISSING`/`POLICY_INVALID`/
  etc. summaries and hints, previously Korean-only) to English across the
  public-facing surface: `pkg/slint` (the public API) and `cmd/slint-gate`
  (the CLI). Internal implementation packages (`pkg/slo/*`, `pkg/kubeutil`,
  `pkg/devutil`) are unaffected — this is a scoped consistency pass on the
  parts an external contributor or judge actually reads/runs, not a
  repo-wide rewrite.

### Added

- CI/quality badges (Tests, Lint, Semgrep, Go Reference, Go version, latest
  release) to `README.md`/`README(Kor).md`.

## [1.5.0] - 2026-07-07

SLI Gate Onboarding UX roadmap (Sprint 1-6): a guided
`init -> inspect -> recommend-policy -> baseline approve/diff/merge -> ci
github-actions -> quickstart` CLI loop that lets a user reach a trustworthy
CI gate without learning the full policy schema first, plus the
`promote_to_fail`/`--exit-on` naming migration. Also ships the
`analyze-dataplane` static analyzer (merged earlier, previously undocumented
under a version header) and 6 custom Semgrep security guardrails, enabled
as blocking CI. See `docs/sli-gate-onboarding-ux.md` and
`docs/security-model.md` for the full design/decision record behind each
item below.

### Added

- `slint-gate analyze-dataplane <manifest-dir>`: new static analyzer for the "dataplane-service" observability contract — reads a directory of Kubernetes YAML manifests (no live cluster) and checks: metrics port naming (`KSL-DP-001`), `/readyz`/`/livez` probe path convention (`KSL-DP-002`), metrics Service/ServiceMonitor wiring (`KSL-DP-004`), and explicit `terminationGracePeriodSeconds` (`KSL-DP-006`). Outputs JSON, SARIF 2.1.0, and a GitHub Actions step summary via `--output-json`/`--output-sarif`/`--github-step-summary`; `--fail-on none|error|warning` controls exit code. CLI-only in this pass — no GitHub composite Action wiring yet.
- `pkg/report`: new generic Finding/Report model (rule ID, severity, message, location) reusable by future dataplane profiles (e.g. a v1.6.0 `dataplane-job` summary gate), plus `WriteJSON`/`WriteSARIF`/`WriteGitHubStepSummary` output writers.
- `pkg/dataplane`: shared, kind-agnostic manifest model (Deployment/StatefulSet/DaemonSet unified as one `Workload` shape, plus `Service`/`ServiceMonitor`) and `LoadDir` directory loader — hand-rolled local structs on top of the existing `gopkg.in/yaml.v3` dependency, no new `k8s.io/**`/`sigs.k8s.io/**` dependency added. `.golangci.yml` gained a `depguard` rule enforcing this for `pkg/dataplane/**`/`pkg/report/**`, mirroring the existing `pkg/slo` core-boundary rule.
- `pkg/dataplane/service`: the dataplane-service checks + a `spec.Registry`-shaped check registry.
- `slint-gate init --profile <name>`: backward-compatible extension of `init` — omitting `--profile` preserves today's exact output. Only `kubebuilder-operator` is supported today; an unrecognized profile is rejected before any file is written. When set, the generated `policy.yaml` records the profile as a comment and `init` prints an `Initialized kube-slint for profile: ...` line.
- `policy.yaml`'s `promote_to_fail` field and CLI/action `--exit-on`/`exit-on`: preferred replacements for `fail_on`/`--fail-on`/`fail-on`, which looked like the same concept despite operating at different layers (policy grade promotion vs. process exit code). See `docs/sli-gate-onboarding-ux.md`'s naming section for the full rationale.
- `slint-gate inspect --summary`: reads a measurement summary and explains, in plain text, which `kubebuilder-operator` profile SLIs were measured (and whether each is threshold/regression-usable or CI-environment-sensitive), which are missing, and overall readiness for a threshold policy/baseline. Read-only — no gate verdict; exits non-zero only if the summary itself can't be loaded.
- `slint-gate recommend-policy --summary --profile --strictness --output [--force] [--dry-run]`: generates a `.slint/policy.yaml` draft from a measured summary — measured profile SLIs become active `thresholds:` entries (each with a `# <reason>` comment), SLIs the profile expects but weren't measured are recorded as comments instead of active rules. `--strictness` (`strict`|`conservative` [default]|`lenient`) governs only the two CI-environment-sensitive SLIs (`rest_client_429_delta`, `workqueue_retries_total_delta`): active-with-relax-comment under `conservative`, active-with-no-comment under `strict`, commented-out under `lenient` — this is deliberately coarse rather than a fake per-rule severity, since `promote_to_fail` is category-wide, not per-threshold. Refuses to overwrite an existing `--output` unless `--force`; `--dry-run` previews to stdout.
- `slint-gate baseline approve --summary --policy --output [--allow-warn] [--force]`: evaluates a summary against a policy (via the same `gate.Evaluate` the top-level gate uses) and, only if it passes, approves it as the known-good baseline for future regression checks. `PASS` is approved by default; `WARN` requires `--allow-warn`; `FAIL`/`NO_GRADE` are always rejected and **cannot** be overridden by `--force` (`--force` only permits overwriting an existing `--output`) — this keeps "measurement failure never produces a false approval" intact for baselines the same way it already holds for gate results. Clears `config.evidencePaths` (stale local temp-file paths) before writing; prints a review block before confirming.
- `slint-gate ci github-actions --summary --policy [--baseline] [--action-ref] [--exit-on-mode]`: prints a ready-to-paste GitHub Actions step wired to the given local paths, using the shipped `exit-on` naming. `--action-ref` defaults to the CLI's own build `Version` (e.g. `v1.4.0`) rather than `@main`, matching this repo's existing "pin, don't float" convention for generated snippets. Pure templating — no file I/O or evaluation, so it can be run before the E2E test ever executes.
- `slint-gate baseline diff --baseline --summary [--policy]`: read-only comparison between a stored baseline and a current summary — existing/new/changed/missing SLIs. `--policy` is optional and best-effort: when a metric has a matching threshold rule, a changed value is labeled "improves"/"weakens the known-good baseline" using the same operator-direction logic as `pkg/gate`'s regression check; otherwise the direction is reported as unknown rather than guessed. Never gates — reports `Result: REVIEW_REQUIRED`/`OK`, exits non-zero only on an unreadable baseline/summary file.
- `slint-gate baseline merge --baseline --summary --policy --mode append-new-only [--output]`: safely appends newly-measured SLIs into an existing baseline. Requires the current summary to `PASS` its policy first (no `--allow-warn` equivalent). `append-new-only` never touches an existing SLI's value in either direction — every existing-SLI change, worse or better, is rejected and reported, and SLIs missing from the current summary are left in the baseline rather than deleted. Only `append-new-only` is implemented this pass; other `--mode` values are rejected with a clear "not yet supported" error.
- `kubebuilder-operator` profile expanded from 6 to all 9 real SLIs `pkg/slint.BaselineV3Specs()` already defines (`reconcile_success_delta`, `workqueue_adds_total_delta`, `rest_client_requests_total_delta` are newly included) as a new `informational` candidate tier — shown in `inspect`/`recommend-policy` when measured, but never promoted to an active threshold at any `--strictness`, since they're raw activity counters with no principled pass/fail value. No second built-in profile was added: every other profile name previously floated in the onboarding doc (`dataplane-service`, etc.) has no backing spec/collector in this codebase, and inventing one risked fabricating metrics that don't exist.
- `slint-gate inspect --profile-file`/`recommend-policy --profile-file`: local custom profile file support (`slint.profile.v1` schema — `candidates: [{id, operator, value, tier, reason}]`, deliberately simpler than a per-strictness-override schema, matching how strictness already works here). Resolution order: `--profile-file` (explicit) > `.slint/profiles/<profile-name>.yaml` (repo-local convention) > built-in profile. Validated at load time (schema version, operator, tier, non-empty id) rather than allowed to silently produce an invalid generated policy.
- `slint-gate quickstart [--policy] [--summary] [--baseline]`: read-only status check over the onboarding artifacts (policy/summary/optional baseline) and a single "Next:" suggestion for what to run — reuses `gate.Evaluate`/`summary.LoadFile` exactly like `inspect`/`baseline approve` already do, no new evaluation logic. Ships as a non-interactive status command rather than the originally-scoped "interactive wizard" — a real stdin-prompted CLI is a different kind of work from anything else in this tool (harder to test, needs TTY/non-interactive-CI handling), and that scoping question got no response in time, so the lower-risk option shipped; a true interactive flow remains possible later if requested.
- `recommend-policy`: when an active rule's own default operator/value is already violated by the currently measured value, an extra `# ⚠ measured value (...) does not satisfy this default threshold` comment is added under the rule. Deliberately does not auto-adjust the threshold to fit the observed sample.
- `.semgrep/rules/`: 6 custom Semgrep rules implementing `docs/security-model.md`'s previously-unimplemented "Static Guardrail Plan" (`kube-slint-no-direct-service-url-format`, `kube-slint-no-bearer-token-in-curl-args`, `kube-slint-no-insecure-skip-verify`, `kube-slint-no-clusterrolebinding-default`, `kube-slint-no-stat-before-write`, `kube-slint-no-unsafe-cleanup`), each with a paired positive/negative Go test fixture (`semgrep --test .semgrep/rules`/`make semgrep-test`). Enabled as blocking CI (`.github/workflows/semgrep.yml`, `make semgrep` locally) — the real codebase was scanned and found fully compliant after adding `// nosemgrep` (bare, not rule-id-qualified — directory-based `--config` loading namespaces rule IDs by path, which is invocation-dependent) at two already-accepted call sites (the existing `--output`/`--baseline` overwrite-refusal checks; `sweep.go`'s label-filtered-then-delete-by-name cleanup) and excluding `pkg/kubeutil/rbac.go`'s dead/test-only `ApplyClusterRoleBinding` helper wholesale via `.semgrepignore`.

### Deprecated

- `policy.yaml`'s `fail_on` (use `promote_to_fail`), CLI `--fail-on` (use `--exit-on`), and the GitHub Action's `fail-on` input (use `exit-on`). All three still work — the old and new names are unioned/take precedence per `docs/gate-contract.md`'s `exit-on` Modes section — and using the deprecated names produces a non-fatal notice (a `policy_warnings` entry for `fail_on`, a stderr line for `--fail-on`). No removal date set yet.

### Security

- `pkg/slo/fetch/curlpod`: `ValidateMetricsURL` and `isDangerousNamespace` — `ServiceURLFormat` is now validated before any curl pod is created. Default-deny: external hosts, unsupported URL schemes, malformed service/namespace values, and `kube-system`/`kube-public`/`kube-node-lease` target namespaces are all rejected unless explicitly opted into via new `Dangerously*` fields on `SessionConfig`/`curlpod.Client`/`CurlPod` (`DangerouslyAllowExternalMetricsURL`, `DangerouslySkipTLSVerify`, `DangerouslyAllowKubeSystemNamespace`). A rejection surfaces as a normal fetch error → `CollectionStatus=Failed` → `NO_GRADE`, not a panic or silent pass. See `docs/security-model.md`.
- `pkg/slo/fetch/curlpod/client.go`: `curlpod.New()`'s `TLSInsecureSkipVerify` default changed from `true` ("defaulting to true for backward compatibility with E2E suite") to `false` — this contradicted `docs/security-model.md`'s default-deny policy. `TLSInsecureSkipVerify` is now deprecated in favor of `DangerouslySkipTLSVerify` (same effect, OR'd for compatibility).
- `cmd/slint-gate/init.go`'s onboarding snippet no longer sets the now-deprecated `TLSInsecureSkipVerify: true` by default; it's now a commented-out `DangerouslySkipTLSVerify: true` line with a note on when to use it.
- `pkg/slo/summary/schema.go`: `Validate` now also rejects duplicate result IDs and unrecognized result statuses (previously only checked schema version, `generatedAt`, and empty IDs). `pkg/gate/gate.go`'s `loadMeasurement` now calls the fuller `Validate` (was only `ValidateSchemaVersion`), so these join malformed JSON as `MEASUREMENT_INPUT_CORRUPT`/`NO_GRADE` instead of being silently accepted.
- `pkg/gate/gate.go`'s `validatePolicy`: rejects duplicate threshold names, a NaN threshold value, and a negative `regression.tolerance_percent`.
- `pkg/gate/testdata/{summary,policy}/` + `pkg/gate/badfixtures_test.go`: 16 executable bad-fixture tests per `docs/test-strategy.md`'s Bad Fixture Matrix, asserting invalid summary/policy input never produces `PASS`.

### Fixed

- `.gitignore`: a bare `slint-gate` pattern unintentionally matched the `cmd/slint-gate` source directory (not just an accidental root-level binary build), forcing `git add -f` on every new file under it. Anchored to `/slint-gate`.

### Removed

- `KSL-DP-003` (probe wiring) and `KSL-DP-005` (resource requests/limits) checks — confirmed exact duplicates of kube-linter's actively-maintained `no-liveness-probe`/`no-readiness-probe` and `unset-cpu-requirements`/`unset-memory-requirements` checks. `analyze-dataplane` now only implements checks not already covered by established manifest linters; pair it with kube-linter (or similar) for probe/resource hygiene.

## [1.4.0] - 2026-07-04

Post-RC hardening sprint: gate reliability/regression correctness, secret
redaction coverage, fetcher metric normalization, and moving gate evaluation
to a public package (`pkg/gate`). See `docs/post-rc-hardening-design.md` for
the full before/after analysis behind each item below.

### Changed

- `pkg/gate/gate.go`: `reliability.collectionStatus == "Failed"`는 `reliability.required` 설정과 무관하게 무조건 `NO_GRADE`로 승격됨 (기존에는 threshold 규칙이 없고 `reliability.required: false`이면 조용히 `PASS`가 나올 수 있었음). 새 reason 코드 `COLLECTION_FAILED` 추가.
- `pkg/gate/gate.go`: regression 검사가 metric 방향(threshold rule의 `operator`)을 인식함 — `<=`/`<`는 lower-is-better, `>=`/`>`는 higher-is-better로 취급하여 개선(improvement)을 더 이상 회귀로 오탐하지 않음. 방향을 알 수 없는 연산자(`==` 등)는 기존 대칭 tolerance 검사를 유지.
- `pkg/slint/session.go`, `pkg/slint/fetcher_curlpod.go`: curl-pod 기반 fetch(`PreFetch`/`Fetch`)의 외부 context timeout이 더 이상 `ScrapeTimeout`(2분)으로 `WaitPodDoneTimeout`(5분)+`LogsTimeout`을 무효화하지 않음 — `WaitPodDoneTimeout+LogsTimeout+여유`로 계산.
- `pkg/slint/sweep.go`: orphan sweep 제외 셀렉터(`slint-run-id!=...`)가 다른 셀렉터들과 동일하게 `SanitizeKubernetesLabelValue`를 거침.
- `pkg/slint/attach.go`: `SessionConfig.Token`이 비어 있어도 더 이상 테스트가 실패하지 않음 — 기본 curlpod fetcher는 pod에 마운트된 ServiceAccount 토큰을 사용하므로 `Token` 필드는 커스텀 Fetcher를 위한 호환성 필드로만 남음.
- `pkg/slo/fetch/curlpod/client.go`: 생성되는 curl pod PodSpec에 `automountServiceAccountToken: true`를 명시 — ServiceAccount 기본값에 의존하지 않음.
- `cmd/slint-gate/diagnose.go`: `POLICY_INVALID` 진단 힌트에 `schema_version`/`fail_on`/`reliability.min_level` 원인을 명시 (기존에는 YAML 문법과 operator만 언급해 원인을 못 찾기 쉬웠음).
- `examples/kind-hello-operator/manifests/rbac.yaml`: `ClusterRole`/`ClusterRoleBinding` → 네임스페이스 스코프 `Role`/`RoleBinding`으로 변경 (`slint-gate init --emit-rbac` 템플릿과 정합).
- `pkg/slo/fetch/promtext`: bare-name 메트릭 합산 로직(`Aggregate`/`ParseTextToMapWithAggregates`)을 curlpod fetcher 전용 코드에서 공용 패키지로 이동하여 curlpod/portforward fetcher가 동일한 metric 의미를 갖도록 통일. 실제 unlabeled series가 있으면 덮어쓰지 않고, histogram bucket(`le` 레이블)/summary quantile(`quantile` 레이블)은 합산 대상에서 제외하도록 개선.
- `pkg/slint/session.go`: `Session.End()`가 세션이 직접 생성한 fetcher에만 `Stop()`을 호출함 — `SessionConfig.Fetcher`로 사용자가 직접 공급한(여러 세션에서 재사용할 수 있는) fetcher는 더 이상 첫 `End()` 호출로 강제 종료되지 않음.
- `.github/workflows/slint-gate.yml`: `workflow_dispatch` 기본값이 항상 PASS하는 데모 fixture를 가리킨다는 점을 주석과 input 설명에 명시.
- `internal/gate` → `pkg/gate`: gate 평가 로직을 공개 패키지로 이동 — 같은 모듈 밖의 소비자(향후 MCP 서버 등)가 CLI와 동일한 gate 판단 로직을 재사용할 수 있게 됨. import 경로만 바뀌었고 동작은 동일. `.golangci.yml`의 `internal/*` dupl/lll 예외 규칙을 `pkg/gate/*`로, 관련 워크플로우의 path filter(`internal/gate/**`)를 `pkg/gate/**`로 갱신. `Dockerfile`의 이제 존재하지 않는 `COPY internal/ internal/` 라인 제거.

### Security

- `pkg/slo/evidence/redact.go`: 시크릿 redaction 패턴이 `Bearer <token>`/`key=value` 형태 외에 JSON-quoted(`"token": "..."`), CLI 플래그(`--token`, `--client-key-data`, `--certificate-authority-data`), YAML/plain-colon(`token: ...`) 형태도 커버하도록 확장. `serviceAccountToken`/`clientSecret` 키도 추가로 커버.
- `pkg/kubeutil/token.go`: `requestServiceAccountTokenOnce`가 TokenRequest 응답 JSON 파싱 실패 시 원문 body를 그대로 에러에 포함하던 것을 redact 후 포함하도록 수정 — 손상/잘림된 응답에 남아있는 실제 토큰 조각이 재시도마다 로그로 새는 경로를 차단.

## [1.3.0] - 2026-07-02

### Added

- `test/e2e/harness/harness.go`: backward-compatibility shim — 기존 `test/e2e/harness` import path를 유지하면서 `pkg/slint` 타입·함수를 재노출
- `NOTICE`, `SECURITY.md`, `THIRD_PARTY_LICENSES.md`: Apache 2.0 컴플라이언스 파일 추가
- `docs/demo.md`: 심사위원 대상 PASS/FAIL/NO_GRADE 3단계 데모 가이드
- `docs/competition-readiness-sprint.md`: 공모전 제출 전 완성도 체크리스트
- `examples/kind-hello-operator/Makefile`: `CONTAINER_ENGINE`, `KIND_PROVIDER` 변수 추가 — Docker(기본) 또는 rootless Podman 선택 가능 (`CONTAINER_ENGINE=podman KIND_PROVIDER=podman make demo`)
- `examples/kind-hello-operator/setup.sh`: cgroup v1 조기 감지 및 경고 메시지 출력, `KIND_PROVIDER` env 전달 지원
- `examples/kind-hello-operator/README.md`: cgroup v2 호스트 요구사항 명시, Podman 사용법 추가

### Changed

- `pkg/slint/*`: `test/e2e/harness` 패키지를 `pkg/slint`로 이동 (공개 import path 확정)
- CI: `golangci-lint-action@v9`, `actions/checkout@v6`, `actions/setup-go@v6`, `actions/upload-artifact@v7` 업그레이드
- `examples/kind-hello-operator/operator/Dockerfile`: `GO111MODULE=off` 추가 (stdlib-only 빌드 안정화)
- `examples/kind-hello-operator/e2e/e2e_test.go`: `--fail-on` 플래그 값을 `FAIL_OR_NOGRADE`로 수정

### Fixed

- `.gitignore`: `slint-gate-summary.json` 생성 artifact 제외 추가

## [1.2.0] - 2026-06-02

### Added

- `pkg/slo/fetch/k8sobject`: `K8sObjectFetcher` — `fetch.SnapshotFetcher` 구현체. kubectl list 기반으로 Pod/Job 오브젝트 수를 캡처하며 기존 2점 엔진 모델과 호환됨. `ExcludeSelector`로 curlpod 등 kube-slint 관리 리소스를 측정 대상에서 제외 가능
- `K8sObjectFetcher` 계산 메트릭: `{prefix}_count` (총 오브젝트 수), `{prefix}_orphan_end` (ownerRef 없는 오브젝트), `{prefix}_ownerref_missing_end` (ownerRef UID가 현재 셋에 없는 오브젝트), `{prefix}_stuck_terminating_end` (설정 임계값 초과 Terminating 오브젝트)
- `pkg/slo/spec/jumi_churn.go`: `JUMIChurnSpecs()` — JUMI K8s 오브젝트 churn 측정용 SLI 스펙 세트 (jobs/pods created delta, orphan, ownerref_missing, stuck_terminating 종단 게이지)

## [1.1.0] - 2026-06-01

### Added

- `internal/gate`: summary `schemaVersion` 검증 — 비어 있거나 미지원 버전이면 `MeasurementStatus=unsupported_schema`, `GateResult=NO_GRADE`, `Reason=MEASUREMENT_SCHEMA_UNSUPPORTED` 반환
- `pkg/slo/summary`: `SchemaVersion` 상수, `ValidateSchemaVersion()`, `Validate()`, `LoadFile()`, `WriteFile()` 공개 — 외부 도구가 별도 struct 없이 summary contract를 사용할 수 있도록 함
- `docs/integration/summary-schema.md`: 최소·전체 JSON 예시, Go API 사용법, status 표, CLI contract
- `internal/gate`: `runResultStatus()` — 엔진이 계산한 SLI 상태(`fail`/`block`→FAIL, `warn`→WARN, `skip` 무값→NO_GRADE)를 gate 평가에 반영; `result_status` check 카테고리 및 `RESULT_STATUS_FAIL` reason 추가
- `pkg/slo/spec`: `CounterResetPolicy` 타입 (`warn`/`no_grade`/`fail`/`skip`) + `ComputeSpec.OnCounterReset` 필드 — ComputeDelta에서 delta<0 처리 정책을 SLI별로 설정 가능
- `pkg/slo/evidence`: `RedactString()` / `RedactMap()` — Bearer 토큰, `token=`/`password=`/`secret=` 값 마스킹 유틸리티
- `examples/consumer-specs/jumi-ah/specs.go`: JUMI Phase 1 handoff gRPC 클라이언트 카운터 및 K8s 스포너 라이프사이클 SLI 스펙 추가
- `docs/curlpod-security.md`: 최소 RBAC, NetworkPolicy 예시, Pod 식별 레이블, cleanup 실패 대응 절차
- `docs/verification-sources.md`: Tier 1(현재 2점 엔진)/Tier 2(엔진 확장 필요) source 모델 설계 경계 문서; `WindowFetcher` 인터페이스 초안

### Changed

- `pkg/slo/spec/jumi_ah_minimum.go`: `jumi_jobs_created_delta`, `jumi_fast_fail_trigger_delta` — `OnCounterReset: CounterResetNoGrade` 적용 (counter reset 시 promotion 차단)
- `pkg/slo/fetch/curlpod`: `CurlPod.Run()` — 파드 삭제 실패를 조용히 무시하던 코드를 경고 로그 출력으로 교체 (namespace/podName/error/selector 포함)
- `pkg/slo/engine`: 하드코딩된 `"slo.v3"` → `summary.SchemaVersion` 상수 참조

## [0.1.0] - 2026-05-11

### Added

- `pkg/slint`: 안정적 공개 API 패키지 (`Session`, `SessionConfig`, `NewSession`, `DefaultSpecs`, `BaselineSpecs` type aliases)
- `pkg/slint/token.go`: `ReadServiceAccountToken`, `ReadServiceAccountTokenFromEnv` 온보딩 헬퍼
- `SessionConfig.ServiceURLFormat`: 메트릭 URL 포맷 오버라이드 필드; `slint.ServiceURLHTTPS` / `slint.ServiceURLHTTP` 상수
- `cmd/slint-gate`: `--fail-on` 플래그 (`NEVER`|`FAIL`|`FAIL_OR_WARN`|`FAIL_OR_NOGRADE`|`FAIL_WARN_OR_NOGRADE`); 기본값 `NEVER`
- `.github/actions/slint-gate`: GitHub Composite Action, 4단계 fail-on 지원, artifact upload, step summary 렌더링
- `internal/gate`: policy.yaml unknown field 감지 → `PolicyWarnings` in Summary JSON + stderr 경고
- `examples/kind-hello-operator`: kind 클러스터 기반 end-to-end 예제 (stdlib-only 메트릭 서버, 매니페스트, RBAC, E2E 테스트, policy)
- `examples/consumer-specs/jumi-ah/specs.go`: JUMI→AH 데이터플레인 consumer spec 예제
- `LICENSE`: Apache 2.0
- `CONTRIBUTING.md` + GitHub issue 템플릿 (bug, feature)

### Changed

- `workqueue_depth_end`: `ComputeSingle` → `ComputeEnd` (이름과 실제 동작 일치)
- `Session.End()`: dual-write 전략 (unique 파일 + `artifacts/sli-summary.json` static alias)
- `Dockerfile`: `golang:1.25` + `distroless/static:nonroot`, `cmd/slint-gate` CLI 이미지 빌드
- `hack/prepare-baseline-update.sh`: Python/pyyaml 완전 제거 → `go run ./cmd/slint-gate` + jq 기반 재작성

### Fixed

- `slint-gate` action.yml: CLI의 action 컨텍스트 exit 1 충돌 수정; fail-on 결정권을 bash step으로 이전
- kind 예제 policy.yaml: metric ID를 `sli-summary` `results[].id`와 일치하도록 수정
- kind 예제 artifacts 경로 및 slint-gate 상대 경로 수정

### Removed

- `hack/slint_gate.py`: Python gate 프로토타입 삭제

[Unreleased]: https://github.com/HeaInSeo/kube-slint/compare/v1.2.0...HEAD
[1.2.0]: https://github.com/HeaInSeo/kube-slint/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/HeaInSeo/kube-slint/compare/v1.0.1...v1.1.0
[0.1.0]: https://github.com/HeaInSeo/kube-slint/releases/tag/v0.1.0
