# SLI Gate Onboarding UX

Date: 2026-07-05
Status: Initial technical design note

## Purpose

This document defines the UX problem and target design for helping a new
kube-slint user choose SLIs, generate an initial policy, establish a baseline,
and wire the result into CI.

kube-slint's product center is not generic Kubernetes linting. It is a CI
guardrail for Kubernetes operational SLI regressions observed during tests.

The user-facing onboarding question is:

```text
How does a new project go from "I have an E2E test and /metrics" to
"I have a trustworthy SLI regression gate in CI" without learning every
internal kube-slint concept first?
```

## Current UX Rating

Estimated current score for new-project SLI/gate onboarding:

```text
6.5 / 10
```

The core model is solid, but self-service onboarding still requires too much
manual judgment.

Strong points:

- `pkg/slint` exposes a consumer-facing `Session` API.
- `slint.DefaultSpecs()` gives kubebuilder/operator users a usable starting
  SLI set.
- `.slint/policy.yaml` is the preferred policy path.
- `slint-gate` has a clear `PASS | WARN | FAIL | NO_GRADE` model.
- GitHub Action and kind demo paths exist.
- Namespace-scoped RBAC and token handling are clearer after post-RC hardening.

Friction points:

- Users must decide which SLIs matter for their project.
- Users must hand-tune thresholds before they understand normal metric ranges.
- Baseline creation and approval flow is conceptually heavy.
- `policy.fail_on` and CLI/action `--fail-on` are two separate layers.
- `NO_GRADE` is correct but initially unfamiliar.
- Kubernetes details such as metrics Service, ServiceAccount, RBAC,
  ServiceURLFormat, and TLS settings appear early.
- kube-slint does not yet provide a guided "inspect -> recommend -> baseline
  -> CI" loop.

## Target UX Score

Target onboarding score:

```text
9 / 10
```

10/10 requires long-term maturity such as release binaries, dogfooding,
schema migration policy, full negative E2E coverage, and supply-chain
attestation. The onboarding UX itself should aim for 9/10 first.

## Target User Journey

The ideal first-time workflow:

```text
1. User runs existing E2E test with kube-slint attached.
2. kube-slint writes artifacts/sli-summary.json.
3. User runs an inspect command.
4. kube-slint explains which SLIs were measured and which were missing.
5. User runs a recommend command.
6. kube-slint generates a conservative policy draft.
7. User approves the first healthy run as a baseline.
8. kube-slint prints CI YAML.
9. CI blocks SLI regression or untrustworthy measurement.
```

The user should not need to know the full policy schema before seeing a useful
first gate.

## Proposed CLI Flow

### 1. Initialize

```sh
slint-gate init --profile kubebuilder-operator
```

`--profile` is a backward-compatible extension of the existing `init`
command, not a new command. Omitting `--profile` preserves today's exact
`init` output and behavior. Only `kubebuilder-operator` is supported today;
an unrecognized profile name is rejected with a clear error before any file
is written. When a profile is given, `init` prints a leading
`Initialized kube-slint for profile: <name>` line and records the profile
as a comment in the generated `policy.yaml`.

Expected output:

- `.slint/policy.yaml` draft;
- namespace-scoped RBAC manifest, if requested;
- code snippet using `pkg/slint`;
- next command suggestions.

UX goal:

```text
Give the user a safe starting point, not a blank policy file.
```

### 2. Inspect Summary — implemented (Sprint 2)

```sh
slint-gate inspect --summary artifacts/sli-summary.json
```

Actual output:

```text
Measured shift-left SLIs:
  reconcile_total_delta            14           usable for threshold + regression
  reconcile_error_delta            0            usable for threshold + regression
  workqueue_depth_end              0            usable for threshold + regression
  rest_client_5xx_delta            0            usable for threshold + regression
  rest_client_429_delta            0            usable, but may be CI-environment sensitive
  workqueue_retries_total_delta    0            usable, but may be CI-environment sensitive

Missing profile SLIs:
  (none)

Readiness:
  Threshold policy: ready
  Baseline approval: ready
  Regression gate: not enabled yet
  Measurement confidence: complete

Next:
  slint-gate recommend-policy --summary artifacts/sli-summary.json --profile kubebuilder-operator
```

When a profile SLI is absent from the summary, it's listed under "Missing
profile SLIs" with a keep-commented-out recommendation instead. `inspect`
never produces a gate verdict — it exits non-zero only if the summary file
itself can't be loaded (missing, malformed, or unsupported schema version).

UX goal:

```text
Explain what kube-slint saw before asking the user to write policy.
```

### 3. Recommend Policy — implemented (Sprint 2)

```sh
slint-gate recommend-policy \
  --summary artifacts/sli-summary.json \
  --profile kubebuilder-operator \
  --strictness conservative \
  --output .slint/policy.yaml
```

Actual behavior:

- measured SLIs become active `thresholds:` entries; SLIs the profile expects
  but that are absent from the summary are recorded as trailing comments, not
  active rules — regardless of `--strictness`;
- `--strictness` (`strict` | `conservative` [default] | `lenient`) only
  affects the profile's two CI-environment-sensitive SLIs
  (`rest_client_429_delta`, `workqueue_retries_total_delta`): `strict` makes
  them active with no caveat, `conservative` makes them active with a
  relax-if-flaky comment, `lenient` comments them out entirely (same
  treatment as a missing SLI). The other four SLIs are always active once
  measured, independent of strictness;
- **known limitation, stated rather than faked**: `promote_to_fail` operates
  on the whole `threshold_miss` category, not per-rule, so there is no way
  yet to make one specific threshold WARN-only while others FAIL — that's
  why `lenient` omits a rule entirely instead of downgrading it;
- refuses to overwrite an existing `--output` unless `--force` is passed;
  `--dry-run` prints the draft to stdout instead of writing it;
- defaults to `promote_to_fail: ["threshold_miss"]` with
  `# - "regression_detected"` commented out, and `regression.enabled: false`
  — matches `init`'s existing template so both commands feel consistent;
- **Sprint 6**: when an active rule's own default operator/value is already
  violated by the currently measured value (e.g. default `<= 0` but the
  measured value is `3`), an extra `# ⚠ measured value (...) does not
  satisfy this default threshold` comment is added directly under the rule.
  This surfaces a real disagreement between the recommendation and reality;
  it deliberately does **not** auto-adjust the threshold to fit the observed
  sample — silently loosening a rule to match one measurement would be the
  kind of unearned adaptivity this project has avoided elsewhere (the tier
  system above, `baseline diff`'s refusal to guess direction without a
  policy).

Example output (`--strictness conservative`, all 6 SLIs measured):

```yaml
schema_version: "slint.policy.v1"

thresholds:
  - name: "reconcile_error_delta_recommended"
    metric: "reconcile_error_delta"
    operator: "=="
    value: 0
    # A healthy operator E2E run should not introduce reconcile errors.

  - name: "rest_client_429_delta_recommended"
    metric: "rest_client_429_delta"
    operator: "=="
    value: 0
    # Client-side throttling may indicate API pressure or controller behavior changes.
    #
    # This SLI can be CI-environment sensitive. If your CI proves this
    # signal is noisy rather than a real regression, consider relaxing this
    # rule only after confirming the source is transient.

regression:
  enabled: false
  tolerance_percent: 10

reliability:
  required: false
  min_level: "partial"

promote_to_fail:
  - "threshold_miss"
  # Uncomment after baseline is established:
  # - "regression_detected"
```

UX goal:

```text
Let users edit a reasonable draft instead of inventing a policy from scratch.
```

### 4. Approve Baseline — implemented (Sprint 3)

```sh
slint-gate baseline approve \
  --summary artifacts/sli-summary.json \
  --policy .slint/policy.yaml \
  --output docs/baselines/my-service-sli-summary.json
```

Actual behavior:

- evaluates the summary against the policy via the same `gate.Evaluate` used
  by `slint-gate` itself (`BaselinePath` empty — this is a first-run
  evaluation against policy only, not a comparison to an existing baseline);
- `PASS` → approved; `WARN` → approved only with `--allow-warn`; `FAIL` and
  `NO_GRADE` → **always rejected — no flag overrides this**, including
  `--force`. `--force` only controls overwriting an existing `--output` file.
  (Letting any flag launder a `FAIL`/`NO_GRADE` result into a "known-good"
  baseline would break the "measurement failure never produces a false
  approval" contract more deeply than any existing `Dangerously*` option
  does — none of those let invalid measurement become a trusted result.)
- clears `config.evidencePaths` before writing (local temp-file paths from
  the original run, meaningless once committed as a baseline) — no other
  normalization is performed;
- prints a review block (paths, gate result, every measured SLI value,
  output path, "Approved.") before confirming.

Rejection example:

```text
Baseline was not approved.

Reason:
  The summary produced NO_GRADE.

How to fix:
  1. Run:
       slint-gate inspect --summary artifacts/sli-summary.json
  2. Review the failed threshold, regression, or reliability check.
  3. Fix the test environment or application behavior and re-run.

Result:
  NO_GRADE
```

UX goal:

```text
Make baseline creation explicit, reviewable, and hard to do accidentally.
```

### 5. Generate CI Snippet — implemented (Sprint 3)

```sh
slint-gate ci github-actions \
  --summary artifacts/sli-summary.json \
  --policy .slint/policy.yaml \
  --baseline docs/baselines/my-service-sli-summary.json
```

Actual output (using the shipped `exit-on` naming, not the deprecated
`fail-on`; `--action-ref` defaults to the CLI's own build `Version`, not
`@main` — consistent with this repo's existing "pin, don't float" convention
for generated snippets):

```yaml
- name: Run kube-slint shift-left SLI gate
  uses: HeaInSeo/kube-slint/.github/actions/slint-gate@v1.5.0
  with:
    measurement-summary: artifacts/sli-summary.json
    policy: .slint/policy.yaml
    baseline: docs/baselines/my-service-sli-summary.json
    exit-on: FAIL_OR_NOGRADE
```

`--baseline` is optional — omitting it emits a threshold-only snippet
without the `baseline:` line. This command does no file I/O beyond stdout
and doesn't evaluate anything; it only needs the path strings, since a user
may run it before ever executing the E2E test just to scaffold CI.

UX goal:

```text
Turn a successful local gate into CI with minimal translation.
```

### 6. Baseline Diff — implemented (Sprint 4)

```sh
slint-gate baseline diff \
  --baseline docs/baselines/my-service-sli-summary.json \
  --summary artifacts/sli-summary.json \
  --policy .slint/policy.yaml
```

Actual behavior:

- read-only/informational, like `inspect` — never gates or exits non-zero
  except when the baseline or summary file itself can't be loaded;
- lists existing baseline SLIs, newly-measured SLIs (always safe to append —
  `Result: OK` if that's the only change), changed existing SLIs, and SLIs
  present in the baseline but missing from the current summary;
- `--policy` is optional and best-effort: when it loads and a metric has a
  matching threshold rule, a changed value is labeled "improves" or "weakens
  the known-good baseline" using the same operator-direction logic as
  `pkg/gate`'s regression check (`<=`/`<` lower-is-better,
  `>=`/`>` higher-is-better). Without a usable policy, or for a metric with
  no rule or a symmetric operator like `==`, the direction is reported as
  unknown rather than guessed from the metric name;
- `Result: REVIEW_REQUIRED` whenever anything changed or went missing;
  `Result: OK` otherwise.

UX goal:

```text
Show whether the baseline is stale before touching it.
```

### 7. Baseline Safe Merge — implemented (Sprint 4, `append-new-only` only; `review-existing`/`force-replace` added 2026-07-08)

```sh
slint-gate baseline merge \
  --baseline docs/baselines/my-service-sli-summary.json \
  --summary artifacts/sli-summary.json \
  --policy .slint/policy.yaml \
  --mode append-new-only
```

Actual behavior:

- requires the current summary to `PASS` its policy (via the same
  `gate.Evaluate` `baseline approve` uses) before merging anything — a
  `WARN`, `FAIL`, or `NO_GRADE` summary is rejected outright, no `--allow-warn`
  equivalent for this command;
- `--baseline` must already exist (`baseline approve` first if it doesn't);
  `--output` defaults to `--baseline` itself (in-place update);
- **`append-new-only` (default) never touches an existing SLI's value, in
  either direction** — this is the literal meaning of "append-*new*-only."
  Every existing-SLI value difference (worse or better) is listed under
  "Rejected changes" and the baseline keeps its original value; SLIs in the
  baseline but absent from the current summary are left in place, never
  deleted; only genuinely new SLI IDs are appended;
- **`review-existing`** (added 2026-07-08, D-023) updates an existing SLI's
  value only when it's a confirmed improvement in the direction implied by
  `policy.yaml`'s threshold operator for that metric (reusing
  `gate.LowerIsBetter`/`gate.HigherIsBetter`, same as `baseline diff`'s
  improve/weaken wording). A change with no recognized direction, or a
  regression, is still rejected and left unchanged — this mode never
  guesses;
- **`force-replace`** (added 2026-07-08, D-023) unconditionally overwrites
  every existing SLI whose value changed, regardless of direction. This is
  a deliberate rebaselining escape hatch, not a default-safe mode — Sprint
  4's original doc flagged it as the more dangerous of the two deferred
  modes, and that judgment still holds; it's just no longer *unimplemented*,
  it's opt-in via `--mode force-replace`;
- `Result: MERGED` (something changed — appended and/or, for
  `review-existing`/`force-replace`, updated), `MERGED_WITH_REJECTIONS`
  (some existing-SLI changes were rejected), or `NO_CHANGE`.

UX goal:

```text
Let the baseline grow safely as the project grows, without silently
weakening or deleting what it already knows.
```

### 8. Quickstart Status — implemented (Sprint 6, descoped from "interactive wizard")

```sh
slint-gate quickstart [--policy] [--summary] [--baseline]
```

Sprint 6 was originally scoped as "interactive wizard." A real stdin-prompted
interactive CLI is a genuinely different kind of work from everything else
in this tool (no existing command reads from stdin or does TTY detection;
it's also harder to unit-test and can misbehave in a non-interactive CI
context). That question was put to the user and got no response in the wait
window, so — consistent with this project's existing pattern of deferring
the riskier/novel option under ambiguous scope (`baseline merge`'s deferred
`force-replace` mode, Sprint 5 declining to fabricate a second built-in
profile) — this shipped as a **non-interactive status command** instead. It
delivers the same practical value (tell the user where they are and what to
run next) without introducing stdin-driven interaction. A true interactive
flow remains possible later if actually requested. (It was — see "Interactive
Wizard" below, added 2026-07-08.)

Actual behavior: a read-only check over the onboarding artifacts (policy
file, measurement summary, optional `--baseline`) that reuses `gate.Evaluate`
and `summary.LoadFile` exactly like `inspect`/`baseline approve` already do
— no new evaluation logic. Prints one line per stage (✓/✗) and a single
"Next:" suggestion, following this precedence: a summary file that exists
but fails to load → `inspect` (it's broken, not just "not run yet"); no
policy and no summary → `init`; no policy → `recommend-policy`; no summary →
run the E2E test; not `PASS` → `inspect`; `PASS` with no approved baseline →
`baseline approve`; baseline present → `ci github-actions`. Baseline
detection is opt-in via `--baseline` (no auto-discovery), since `baseline
approve` itself deliberately has no default output path — baseline
locations are project-specific, and `quickstart` can't invent a convention
that doesn't exist elsewhere in the tool. Never gates; exits non-zero only
on a flag-parsing error.

UX goal:

```text
Answer "where am I and what do I run next?" without re-reading this doc.
```

### 9. Interactive Wizard — implemented (added 2026-07-08, D-024)

```sh
slint-gate wizard [--policy] [--summary] [--baseline]
```

The real stdin-prompted flow Sprint 6 deferred, now built once the previously
unanswered scope question (how does it behave under non-interactive/CI
stdin?) had a concrete answer: refuse to run at all unless stdin is a real
terminal (`golang.org/x/term.IsTerminal`, not a bare `os.ModeCharDevice`
check — `/dev/null` is itself a character device, so that check alone would
let a piped/CI invocation through and then hang forever on the first
prompt). `quickstart` remains the correct choice for CI or scripted use;
`wizard` is for a human sitting at a terminal.

Actual behavior: shares its state detection with `quickstart` (both call the
same `inspectOnboardingState`/`nextOnboardingStep` in
`cmd/slint-gate/onboarding_state.go`, so the two commands can never disagree
about what state the project is in) but, instead of printing one suggested
command, it confirms with the user and then calls the corresponding
subcommand's own unexported `run*(args []string) error` function directly
with programmatically-built args — `runInit`, `runRecommendPolicy`,
`runBaselineApprove`, `runInspect`, `runCIGithubActions` — the same
functions the flag-driven commands use, so there is exactly one
implementation of each step's behavior. Loops until a terminal step
(`ci github-actions`, or a step that requires action outside the CLI's
control, like running the E2E test) or the user declines a confirmation
prompt, either of which stops the loop without error — the user can always
resume by re-running `wizard`.

UX goal:

```text
Same value as quickstart, but for someone who wants the tool to just do it
instead of telling them the command to copy-paste.
```

## Profiles

Profiles should select default SLI specs and policy recommendations.

Initial profile:

```text
kubebuilder-operator
```

Default SLI candidates (Sprint 5: expanded from 6 to all 9 real SLIs
`pkg/slint.BaselineV3Specs()` already defines — see below):

| SLI | Tier | Default recommendation |
|---|---|---|
| `reconcile_total_delta` | core | threshold `>= 1` |
| `reconcile_error_delta` | core | threshold `== 0` |
| `workqueue_depth_end` | core | threshold `<= 0` |
| `rest_client_5xx_delta` | core | threshold `== 0` |
| `rest_client_429_delta` | noisy | threshold `== 0`, `--strictness`-governed |
| `workqueue_retries_total_delta` | noisy | threshold `== 0`, `--strictness`-governed |
| `reconcile_success_delta` | informational | shown, never gated (no principled threshold) |
| `workqueue_adds_total_delta` | informational | shown, never gated |
| `rest_client_requests_total_delta` | informational | shown, never gated |

**Why only one built-in profile**: every other profile name this doc
originally floated (`dataplane-service`, `controller-runtime-operator`,
`custom-prometheus`, etc.) has no backing SLI spec or collector anywhere in
this codebase. Fabricating a second built-in profile would mean inventing
metric names with nothing behind them — a real honesty risk for what's
meant to be a shift-left guardrail grounded in actually-measured signals.
Instead, Sprint 5 closed the gap between `kubebuilder-operator`'s 6
candidates and `BaselineV3Specs()`'s 9 real specs, and added local custom
profile support (below) as the extensibility path for anything else.

### Local Custom Profile Files (Sprint 5)

Resolution order for both `inspect --profile`/`--profile-file` and
`recommend-policy --profile`/`--profile-file`:

```text
1. --profile-file <path>                      (explicit, always wins)
2. .slint/profiles/<profile-name>.yaml        (repo-local convention)
3. built-in profile lookup by --profile name  (today: kubebuilder-operator only)
```

Custom profile schema (`slint.profile.v1`) — deliberately simpler than the
per-strictness-override schema floated earlier in this doc, since
`recommend-policy`'s strictness logic operates on a global `--strictness`
flag plus a per-candidate tier, not per-rule per-strictness value maps:

```yaml
schema_version: "slint.profile.v1"
name: "my-custom-profile"
description: "..."
candidates:
  - id: "some_metric_delta"
    operator: "=="        # <, <=, >, >=, == (omit entirely for tier: informational)
    value: 0
    tier: "core"           # core | noisy | informational (default: core)
    reason: "..."
```

Validated at load time (unsupported `schema_version`, unknown `operator`,
unrecognized `tier`, or an empty `id` are all rejected with a clear error)
rather than allowed to silently produce an invalid generated `policy.yaml`.
A pinned remote profile registry remains future work, as originally staged.

## Naming: policy `promote_to_fail` vs CLI/action `--exit-on`

`policy.fail_on` and the CLI/action `--fail-on`/`fail-on` looked like the
same concept because both contain the word "fail," but they operate at
different layers:

```text
policy.promote_to_fail (policy.yaml)
  Which gate conditions (threshold_miss, regression_detected) are promoted
  from WARN to FAIL. Decided by kube-slint's own gate evaluation.

CLI/action --exit-on / exit-on
  Which gate_result values (FAIL, WARN, NO_GRADE) cause the process/job to
  exit non-zero. Decided by the caller (CI), not by kube-slint.
```

To make this split visible in the names themselves — rather than relying on
documentation, matching this project's existing `Dangerously*` naming
convention — the policy field is now `promote_to_fail` and the CLI/action
flag is now `--exit-on`/`exit-on`.

`fail_on`, `--fail-on`, and the action's `fail-on` input already shipped in
tagged releases (v1.0.0–v1.4.0), so this is a **dual-support** migration, not
a breaking rename:

- `policy.yaml`'s `fail_on` and `promote_to_fail` are unioned — either field
  (or both) is honored. Using `fail_on` adds a non-fatal deprecation entry to
  `slint-gate-summary.json`'s `policy_warnings`.
- The CLI's `--exit-on` wins if both `--exit-on` and `--fail-on` are passed;
  `--fail-on` alone still works but prints a one-line stderr deprecation
  notice.
- The GitHub Action's `exit-on` input wins if both `exit-on` and `fail-on`
  are set; `fail-on` alone still works unchanged (its existing default,
  `FAIL_OR_NOGRADE`, is preserved).

There is no removal date yet for the deprecated names.

## UX Concepts To Hide Until Needed

The first-run path should avoid front-loading:

- schema compatibility details;
- every possible `fail-on` mode;
- custom SLI spec authoring;
- ServiceURLFormat override;
- dangerous TLS settings;
- MCP integration;
- supply-chain/release details.

Those topics should remain documented, but the first-time flow should only
surface them when the user's environment requires them.

## Error Message Requirements

Failures should answer four questions:

```text
What happened?
What does it mean for my test/gate?
How do I fix it?
What is the gate result?
```

Example:

```text
kube-slint could not recommend a policy.

Reason:
  The summary has no measured SLI results.

What this means:
  Your E2E test may have passed, but kube-slint did not collect enough
  operational signal to build a gate.

How to fix:
  1. Confirm the metrics Service name.
  2. Confirm the Service exposes /metrics.
  3. Run slint-gate inspect --summary artifacts/sli-summary.json.

Result:
  NO_GRADE
```

## Acceptance Criteria

A new-project onboarding UX is acceptable when:

- a user can generate a starter policy from a valid summary;
- a user can inspect missing SLIs without reading Go structs;
- a user can create a baseline through an explicit approval command;
- CI YAML can be generated from the same paths used locally;
- invalid summary or policy never produces `PASS`;
- `NO_GRADE` is explained as untrustworthy measurement, not app test failure;
- default flow does not require ClusterRoleBinding;
- default flow does not require external metrics URLs;
- dangerous options are not presented as normal quickstart knobs.

## Non-Goals

- Replacing the user's E2E assertions.
- Replacing Prometheus.
- Providing full SLO management.
- Automatically deciding production SLOs.
- Enabling write-capable AI/MCP actions.

## Implementation Handoff

Recommended implementation order:

1. ~~`slint-gate inspect --summary`.~~ Done (Sprint 2).
2. ~~`slint-gate recommend-policy --summary --profile`.~~ Done (Sprint 2).
3. ~~`slint-gate baseline approve`.~~ Done (Sprint 3).
4. ~~`slint-gate ci github-actions`.~~ Done (Sprint 3).
5. Docs update for quickstart and troubleshooting.

Each command should be independently useful and testable.

## Open Decisions

- Whether first-run baseline absence remains `WARN` or becomes configurable
  as `NO_GRADE`.

Resolved:

- **CI snippet action target**: current local composite action
  (`.github/actions/slint-gate`), pinned via `--action-ref` (defaults to the
  CLI's own build `Version`, e.g. `v1.5.0` — never `@main`). Revisit once a
  release-binary-based action exists.
- **`baseline approve` grade gate**: `PASS` approved; `WARN` approved only
  with `--allow-warn`; `FAIL`/`NO_GRADE` always rejected, with no flag
  (including `--force`) able to override — `--force` only controls
  overwriting an existing `--output` file. See "Approve Baseline" above.
- **`baseline merge` mode scope**: all three modes (`append-new-only`,
  `review-existing`, `force-replace`) are implemented as of 2026-07-08
  (D-023). `append-new-only` remains the default — `force-replace` is
  opt-in only, consistent with keeping the more dangerous option out of the
  default surface. See "Baseline Safe Merge" above.

- **Profile selection location**: CLI input only for now (`--profile` on
  `init`). The generated `policy.yaml` records the profile as a comment, not
  a schema field, since `slint.policy.v1`'s field compatibility policy isn't
  settled yet. See the "Naming" section above for how `--profile` was wired
  into `init` as a backward-compatible extension.
- **`fail_on` naming**: renamed to `promote_to_fail` (policy) and
  `--exit-on`/`exit-on` (CLI/action), with the old names kept as working,
  deprecated aliases. See "Naming: policy `promote_to_fail` vs CLI/action
  `--exit-on`" above.
- **`recommend-policy` overwrite behavior**: refuses to overwrite an existing
  `--output` by default; `--force` opts in, `--dry-run` previews without
  writing.
- **Threshold strictness default**: mixed strategy, not strict-by-default nor
  comment-only-by-default. Core SLIs (`reconcile_total_delta`,
  `reconcile_error_delta`, `workqueue_depth_end`, `rest_client_5xx_delta`) are
  always active once measured; the two CI-environment-sensitive SLIs
  (`rest_client_429_delta`, `workqueue_retries_total_delta`) are governed by
  `--strictness` (default `conservative`: active with a relax-if-flaky
  comment). See "Recommend Policy" above for the exact per-strictness
  behavior and the stated `promote_to_fail`-is-category-wide limitation.
