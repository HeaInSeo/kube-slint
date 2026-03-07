# slint-gate spec (draft, 2026-03-07)

## 1) Purpose and Scope

`slint-gate` is a policy evaluation layer over measurement outputs.

What it does:
- evaluates policy outcomes from existing measurement summaries
- applies two gate axes:
  - absolute threshold checks
  - regression checks vs baseline
- returns an explicit gate result enum:
  - `PASS`, `WARN`, `FAIL`, `NO_GRADE`

What it does not do:
- does not run correctness tests (`test/lint/mock-e2e`)
- does not replace measurement generation logic
- does not parse human prose docs for automation input

Difference from correctness testing:
- correctness testing validates implementation behavior
- `slint-gate` validates operational quality policy outcomes

Difference from `roadmap-status`:
- `roadmap-status` = visibility of project stage/progress metadata
- `slint-gate` = pass/warn/fail/no-grade decision on SLI policy evaluation

## 2) Input Contract

### 2.1 measurement result input

- primary input:
  - `sli-summary.json` (or schema-equivalent summary output)
- current repo reality:
  - summary output exists via harness/session path
  - path/layout may vary by runner/job context
- requirement:
  - input file must be readable and schema-valid enough to evaluate policy

### 2.2 policy input

- policy contains:
  - absolute threshold rules
  - regression rules
  - reliability minimum requirements (if enabled)
- candidate path (draft):
  - `docs/policy/slint-gate-policy.yaml` (proposed, not implemented)
  - or CI-provided policy file path via env/input (proposed)
- status:
  - draft contract only, concrete file path not finalized

### 2.3 baseline input

- baseline is optional for first-run, first-class when available
- baseline source candidates (draft):
  - artifact from previous run
  - repository-stored baseline file
  - external store path configured by CI
- status:
  - source selection is not finalized in this phase

## 3) Output Contract

### 3.1 gate result enum

- `PASS`: policy evaluation completed and no policy violation found
- `WARN`: policy evaluation completed but non-blocking quality risk detected
- `FAIL`: policy violation detected (CI-failable)
- `NO_GRADE`: policy decision unavailable (insufficient comparison/evaluation context)

### 3.2 machine-readable output (draft)

- output file candidate:
  - `slint-gate-summary.json`
- minimal fields (draft):
  - `gateResult`
  - `reasonCodes`
  - `measurementStatus`
  - `policyStatus`
  - `baselineStatus`
  - `timestamp`

### 3.3 human-readable output

- summary target:
  - GitHub Actions Step Summary and/or console output
- expected content:
  - gate result
  - absolute threshold decision
  - regression decision
  - reliability decision
  - baseline availability state
  - CI fail/continue rationale

### 3.4 CI failure rule

- CI may fail when:
  - gate result is `FAIL`
- CI should not fail by default when:
  - gate result is `WARN` or `NO_GRADE` (policy-dependent, default non-blocking)

## 4) First-run and Baseline Handling

### baseline missing

- first-run default proposal:
  - evaluate absolute thresholds
  - skip regression comparison
  - default gate result:
    - `PASS` if absolute thresholds pass and reliability acceptable
    - `WARN` if absolute thresholds pass but regression could not be evaluated
    - `FAIL` if absolute thresholds fail

### baseline present

- evaluate:
  - absolute thresholds
  - regression vs baseline
- regression violation defaults to `FAIL`

### baseline corrupt/unreadable

- classify as:
  - policy evaluation unavailable for comparison axis (not measurement failure by itself)
- default result:
  - `WARN` or `NO_GRADE` (proposed default: `NO_GRADE` when regression is required)

## 5) Decision Model (proposed defaults)

1. absolute threshold miss:
   - default: `FAIL`
2. regression vs baseline detected:
   - default: `FAIL`
3. reliability/skew insufficient:
   - default: `WARN` (or `NO_GRADE` if policy sets reliability as required gate)
4. measurement input missing/corrupt:
   - default: `NO_GRADE`
5. policy config missing/invalid:
   - default: `NO_GRADE` (configuration issue, no safe policy decision)
6. comparison not possible (baseline unavailable/corrupt):
   - default: `WARN` (first-run) or `NO_GRADE` (comparison-required policy)

## 6) Measurement Failure vs Policy Violation

This is the key separation.

Measurement failure examples:
- measurement input missing
- summary unreadable/corrupt
- required metric absent from measurement output
- reliability data insufficient to trust measurement

Policy violation examples:
- absolute threshold miss
- regression detected against a readable baseline

Why separation matters:
- measurement failure is an evaluation availability problem
- policy violation is a quality decision problem
- gate behavior stays consistent with non-invasive/best-effort philosophy while preserving CI guardrail value

## 7) Minimal decision table (proposed)

- measurement unavailable + no policy evaluation -> `NO_GRADE`
- measurement ok + threshold miss -> `FAIL`
- measurement ok + baseline unavailable + threshold ok -> `WARN` (first-run default)
- measurement ok + regression detected -> `FAIL`
- measurement ok + reliability insufficient -> `WARN` (or `NO_GRADE` if reliability is required gate)
- measurement ok + policy config invalid -> `NO_GRADE`

## 8) hello-operator connection

In `hello-operator`, `slint-gate` is intended to validate:
- whether operator code changes introduce operational SLI regressions early
- whether guardrail feedback is understandable inside ko + Tilt inner-loop flow

Target developer experience:
- after code change, SLI regression signal appears quickly
- developer can distinguish:
  - correctness break
  - measurement issue
  - policy violation

## 9) Status

- This document is a draft/proposed contract for Phase 6-c.
- No workflow/code/baseline storage implementation is included in this step.
