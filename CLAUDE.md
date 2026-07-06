# kube-slint — Claude Context

## 프로젝트 정체성

**kube-slint**는 Kubernetes Operator 개발 시 E2E 테스트 세션에 붙어서 운영 SLI(reconcile rate, workqueue depth, REST client errors)를 수집하고, 선언적 policy로 CI 게이트 판정을 내리는 **shift-left operational SLI guardrail 라이브러리**다.

- correctness 테스트 도구가 아님 (기능이 맞는지 = E2E의 역할)
- 프로덕션 모니터링 시스템이 아님 (point-in-time guardrail)
- operator 코드를 수정하지 않음 (/metrics를 외부에서 scrape)
- 측정 실패는 테스트 실패가 아님 (non-fatal, safety-first)

**최종 목표**: 한국 오픈소스 공모전 제출 및 1등.

**중요**: 이 프로젝트는 공모전 제출용으로만 존재하는 게 아니라 **사내에서 실제로 쓰는 툴**이며, 오픈소스로 공개하면서 그 중 하나로 공모전에도 제출하는 것이다. 따라서 우선순위/설계 판단을 "공모전 제출(마감) 기준으로 충분한가"만으로 결정하면 안 되고, 실사용 운영 도구로서의 완성도(유지보수성, 확장성, 실제 온보딩 경험)도 함께 고려해야 한다. 공모전 마감이 임박했다고 해서 스코프 밖으로 미룬 항목(예: interactive wizard, MCP/IDE 연동, `pkg/policy`/`pkg/summary` 공개 API 정리)이 영구히 불필요하다는 뜻은 아님 — 사내 실사용 관점에서 재검토가 필요할 수 있다.

---

## 현재 상태 (2026-07-07 기준, 갱신)

### 완료된 작업

**Stage 1 — 내부 개발 활용 수준** ✅ (커밋 `141fa7f`, `08eb236`)

| 항목 | 내용 |
|---|---|
| `workqueue_depth_end` | `ComputeSingle` → `ComputeEnd` 수정 + 검증 테스트 |
| 출력 경로 통일 | `Session.End()`가 unique(`sli-summary.<runID>.<testcase>.json`) + static alias(`artifacts/sli-summary.json`) 둘 다 씀 |
| consumer-onboarding 빌드 격리 | `//go:build ignore` 추가 → `go test ./...` 정상화 |
| policy.yaml 정리 | 동작 안 하는 필드(`metadata`, `severity`, `first_run`, `baseline.path`) 제거 |
| Dockerfile 교체 | legacy operator scaffold → `cmd/slint-gate` distroless CLI 이미지 |
| `isEnabledByEnv()` 구현 | `SLINT_ENABLED=0/false`로 비활성화 가능 |
| Dockerfile Go 버전 | `golang:1.24` → `golang:1.25` (go.mod 1.25.5와 일치) |
| Makefile docker-build | slint-gate CLI 이미지 빌드로 연결 |
| dual-write 테스트 | unique + alias 파일 생성/충돌 없음 검증 |
| SLINT_ENABLED 테스트 | 9개 케이스 커버 |

### 다음 작업

**Stage 2 — 오픈소스 배포 수준** (Batch 1 완료)

#### Batch 1 — 오픈소스 기본 요건 ✅ (커밋 `278f1a5`)
- [x] LICENSE 추가 (Apache 2.0)
- [x] `pkg/slint` public API wrapper (`test/e2e/harness` re-export)
- [x] JUMI/AH spec → `examples/consumer-specs/jumi-ah/specs.go` 분리
- [x] NO_GRADE fail-on 옵션 (`--fail-on` flag + action.yml 4-level case 처리)
- [x] CONTRIBUTING.md + GitHub issue 템플릿 (bug, feature)

#### Batch 2 — 개발자 경험 ✅ (커밋 `c005738`)
- [x] kind + hello-operator 예제 (`examples/kind-hello-operator/`)
- [x] Token/ServiceAccount 온보딩 헬퍼 (`pkg/slint/token.go` — ReadServiceAccountToken, ReadServiceAccountTokenFromEnv)
- [x] `ServiceURLFormat` SessionConfig에 노출 + `slint.ServiceURLHTTPS/HTTP` 상수
- [x] policy unknown field 경고 (gate.go — PolicyWarnings in Summary JSON + stderr)

#### Batch 3 — 공모전 완성도 ✅ (커밋 `f0fc563`)
- [x] 한국어 README 보강 (README(Kor).md) — pkg/slint API, --fail-on, token 헬퍼, ServiceURLFormat, kind 예제
- [x] 아키텍처 다이어그램 (docs/architecture.md)
- [x] CHANGELOG (CHANGELOG.md, v0.1.0 엔트리)
- [x] `make coverage` 테스트 커버리지 리포트
- [x] 공모전 제출 문서 (docs/competition-submission.md)
- [x] git tag + GitHub release (v0.1.0 계획은 실제로는 v1.x 시맨틱 버저닝 체계로 대체됨 — `v1.0.0`~`v1.5.0` 태그 존재)

#### Batch 4 — Post-RC Hardening Sprint ✅ (v1.4.0, 커밋 `4862544`~`9172d25`)

`docs/post-rc-hardening-design.md`에 각 항목의 상세 근거/diff 요약 있음.

- [x] gate reliability: `CollectionStatus=Failed`가 `reliability.required` 설정과 무관하게 무조건 `NO_GRADE`로 승격 (R1)
- [x] regression 검사가 threshold rule의 operator로 metric 방향(lower/higher-is-better)을 인식 — 개선을 회귀로 오탐하지 않음 (R2)
- [x] curl-pod fetch의 외부 context timeout이 `WaitPodDoneTimeout+LogsTimeout`을 더 이상 무효화하지 않음 (R4)
- [x] orphan sweep 셀렉터가 다른 곳과 동일하게 RunID를 sanitize (N1)
- [x] `SessionConfig.Token` 필수 검증 제거 + curl pod에 `automountServiceAccountToken: true` 명시 (N2)
- [x] `POLICY_INVALID` 진단 힌트에 `schema_version` 등 실제 원인 명시 (N4)
- [x] 예제 RBAC를 네임스페이스 스코프 `Role`/`RoleBinding`으로 변경 (R5)
- [x] curlpod/portforward fetcher의 metric aggregation을 `pkg/slo/fetch/promtext`로 통일 (R3)
- [x] secret redaction 패턴이 JSON/CLI-flag/YAML 형태까지 커버 (N3)
- [x] `Session.End()`가 세션이 직접 생성한 fetcher에만 `Stop()` 호출 (N5)
- [x] `internal/gate` → `pkg/gate` 이동, `pkg/slint` 진단 로그를 stdout→stderr로 이전 (R6)

N6(workflow demo-fixture 라벨링)도 이번 스프린트에서 완료. 남은 항목: F4(quoted label parser 개선), `pkg/policy`/`pkg/summary` 공개 API 정리(v1.4.0 로드맵 항목 중 미착수분).

**Stage 3 — SLI Gate Onboarding UX 로드맵** ✅ 전체 완료 (Sprint 1~6, 커밋 `b947902`~`54f2112`, v1.5.0에 포함)

`docs/sli-gate-onboarding-ux.md`에 각 항목 상세 설계/결정 근거 있음. 목표: "측정 → 설명 → 추천 → 승인 → CI"를 사용자가 정책 스키마 전체를 배우기 전에 따라갈 수 있는 온보딩 루프 완성.

| Sprint | 내용 |
|---|---|
| 1 (`b947902`) | `slint-gate init --profile` 확장(kubebuilder-operator만, 하위호환). `policy.promote_to_fail`/CLI `--exit-on`/action `exit-on` 신설 — 구 `fail_on`/`--fail-on`/`fail-on`은 dual-support deprecated alias (사용 시 `policy_warnings`/stderr에 마이그레이션 안내). |
| 2 (`f07e064`) | `slint-gate inspect --summary`(읽기전용, 측정/누락 SLI 설명), `slint-gate recommend-policy --summary --profile --strictness`(측정된 SLI만 active threshold, strictness는 noisy 지표 2개에만 적용). |
| 3 (`dad1467`) | `slint-gate baseline approve`(PASS만 기본 승인, FAIL/NO_GRADE는 `--force`로도 절대 승인 안 됨), `slint-gate ci github-actions`(순수 템플릿, `--action-ref` 기본값이 빌드 `Version` 상수). init→inspect→recommend→approve→ci 최소 온보딩 루프 완결. |
| 4 (`3d2db24`) | `slint-gate baseline diff`(정책 있으면 operator 방향으로 improves/weakens 라벨링), `slint-gate baseline merge --mode append-new-only`(기존 SLI 값은 방향 무관 절대 미변경/미삭제; review-existing/force-replace는 스코프 밖). |
| 5 (`6868d51`) | `kubebuilder-operator` 프로파일 6→9개 확장(이미 실재하던 `BaselineV3Specs()` SLI 3개 추가, 새 "informational" tier). `--profile-file`/`.slint/profiles/<name>.yaml` local custom profile 지원. 2번째 built-in 프로파일(`dataplane-service` 등)은 실제 spec이 없어 미제작(지어내지 않음). |
| 6 (`54f2112`) | `slint-gate quickstart`(비대화형 status 명령 — "interactive wizard"는 stdin 프롬프트 등 새로운 위험요소라 스코프 질문에 응답 없어 저위험 대안으로 대체). `recommend-policy`에 threshold-mismatch 경고(⚠, 측정값이 기본 threshold를 이미 위반하면 표시, 자동 조정은 안 함).

**Semgrep 커스텀 가드레일** ✅ (커밋 `06d0cc6`, 온보딩 UX와 별개 트랙, v1.5.0에 포함)

`docs/security-model.md`의 "Static Guardrail Plan"에 계획만 있던 6개 규칙(`kube-slint-no-*`)을 `.semgrep/rules/`에 실제 구현, positive/negative fixture로 검증(`make semgrep-test`), 실제 코드베이스 0 findings 확인 후 CI에 blocking으로 연결(`.github/workflows/semgrep.yml`, `make semgrep`). 이미 받아들여진 패턴 2곳(overwrite-refusal 체크, sweep.go의 label-필터 후 delete-by-name)은 `// nosemgrep`(bare — rule-id 붙이면 디렉토리 기반 config 로딩 시 경로 프리픽스가 붙어 호출 방식에 따라 깨질 수 있음) + 이유 코멘트로 명시적 예외 처리. `pkg/kubeutil/rbac.go`(dead/test-only)는 `.semgrepignore`로 전체 제외.

### 남은 작업 (2026-07-07 기준)

- `baseline merge`의 `review-existing`/`force-replace` 모드 (현재 `append-new-only`만 구현)
- 진짜 interactive wizard (stdin 프롬프트) — Sprint 6에서 비대화형 `quickstart`로 대체, 요청 시 재검토
- IDE/MCP 연동 — 스코프 미확정 스트레치 항목, 별도 트랙(다른 에이전트가 진행하다 보류됨)로 아직 재개 안 됨
- `pkg/policy`/`pkg/summary` 공개 API 정리 (v1.4.0 로드맵부터 미착수)
- F4: quoted label parser 개선
- **v1.5.0 태그/릴리스 완료** (2026-07-07) — Sprint 1-6 온보딩 UX + Semgrep 가드레일 + dataplane-service 분석기 전부 포함
- **1차 코드 제출 마감은 2026-08-15가 아니라 2026-07-17.** (2026-07-07 기준 약 10일 남음 — 이전에 8월 중순으로 잘못 파악하고 있었음, 정정됨). 위 로드맵/가드레일/릴리스가 이미 다 끝나있어서 실제 마감 기준으로도 여유 있는 상태.
- **버퍼 작업(회귀 재확인 + 제출 체크리스트 검토) 완료 (2026-07-07)**: `go build/vet/test`, `gofmt`, `make lint`(golangci-lint 0 issues), `make semgrep-test`(6개 fixture 전부 통과), `make semgrep`(실 코드베이스 0 findings), `bash hack/quality-guardrails.sh`(전체 통과) 재확인. LICENSE/NOTICE/SECURITY.md/THIRD_PARTY_LICENSES.md/CONTRIBUTING.md/CHANGELOG.md 전부 존재 및 go.mod와 정합. README/경쟁 제출 문서에 stale 버전 참조 없음(v1.5.0으로 통일). GitHub Release `v1.5.0`이 draft/prerelease 아닌 정식 published 상태로 `main`을 타겟팅 확인. `kind demo (hello-operator)` 워크플로우가 최신 실제 코드 변경 커밋(v1.5.0 릴리즈 커밋) 기준으로 green 확인. **결론: 제출 관점에서 추가로 급하게 처리해야 할 항목 없음.** (단, 위 "중요" 절에서 밝힌 대로 이 프로젝트는 사내 실사용 툴이기도 하므로, 공모전 제출 완료 이후에도 남은 항목(interactive wizard, MCP/IDE 연동, `pkg/policy`/`pkg/summary` 공개 API 정리, F4, baseline merge 나머지 모드)은 실사용 관점에서 별도로 재검토 필요.)

---

## 핵심 아키텍처

```
test/e2e/harness         ← 소비자 진입점 (현재), pkg/slint로 이동 예정
  Session.Start()        ← 시작 스냅샷 prefetch
  Session.End()          ← 종료 스냅샷 fetch → engine 실행 → JSON 2개 출력

pkg/slo/engine           ← SLI 계산 코어 (ComputeDelta/Start/End)
pkg/slo/spec             ← SLI 스펙 정의 (BaselineV3Specs)
pkg/slo/summary          ← 표준 JSON 출력 스키마
pkg/slo/fetch            ← MetricsFetcher 인터페이스 + curlpod/portforward 구현

pkg/gate                 ← policy 평가 (threshold/regression/reliability), 구 internal/gate
cmd/slint-gate           ← CLI entrypoint
  init                   ← policy.yaml 초안 + RBAC 매니페스트 + SessionConfig 스니펫 생성
  inspect                ← 측정된/누락된 SLI 설명 (읽기전용)
  recommend-policy        ← 측정 기반 policy.yaml 초안 생성 (--strictness, --profile-file)
  baseline approve/diff/merge ← known-good baseline 승인/비교/안전 병합
  ci github-actions        ← GitHub Actions 스텝 스니펫 생성
  quickstart              ← 온보딩 진행 상태 확인 + 다음 명령 제안 (읽기전용)
  analyze-dataplane        ← 정적 매니페스트 분석 (별도 트랙)
.github/actions/slint-gate ← GitHub Composite Action
.semgrep/rules            ← 커스텀 Semgrep 보안 가드레일 (CI blocking)
```

## 출력 파일 구조

```
artifacts/
  sli-summary.<runID>.<testcase>.json  ← 감사 추적용 unique 파일
  sli-summary.json                     ← slint-gate 기본 입력 (latest alias)

slint-gate-summary.json                ← gate 판정 결과
```

## policy.yaml 실제 지원 필드

```yaml
schema_version: "slint.policy.v1"
thresholds:
  - name: string
    metric: string      # sli-summary의 result ID와 일치해야 함
    operator: ">=" | "<=" | ">" | "<" | "=="
    value: float
regression:
  enabled: bool
  tolerance_percent: float
reliability:
  required: bool
  min_level: "partial" | "complete"
promote_to_fail:        # 구 fail_on. 둘 다 union으로 honored, fail_on 사용 시 policy_warnings에 마이그레이션 안내
  - "threshold_miss"
  - "regression_detected"
```

CLI/action 쪽도 동일한 패턴: `--exit-on`/`exit-on`이 신규, `--fail-on`/`fail-on`은 deprecated alias(둘 다 동작, `--fail-on` 사용 시 stderr 경고).

`severity`, `first_run`, `baseline.path`, `regression.mode`, `metadata` 등은 **현재 미지원** (yaml.v3가 조용히 무시함).

## 알려진 기술 부채

| 항목 | 위치 | 설명 |
|---|---|---|
| NextSummaryPath 충돌 파일명 | `session.go:NextSummaryPath` | `file.json-1` 형태 (`.json` 뒤에 suffix) — 미미한 UX 이슈 |

## 주요 결정 히스토리

- **dual-write 전략**: Session.End()는 unique 파일 먼저 쓰고, 성공 시 static alias도 씀. static write 실패는 non-fatal (warning만).
- **isEnabledByEnv**: Attach() 호출 자체가 opt-in. `SLINT_ENABLED=0`으로만 비활성화 가능.
- **consumer-onboarding 격리**: `//go:build ignore`로 go.mod 오염 없이 예제 코드 유지.
- **Dockerfile**: `golang:1.25` builder + distroless/static:nonroot. `IMG=ghcr.io/heainseo/slint-gate:dev`.
