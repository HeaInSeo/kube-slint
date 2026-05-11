# kube-slint — Claude Context

## 프로젝트 정체성

**kube-slint**는 Kubernetes Operator 개발 시 E2E 테스트 세션에 붙어서 운영 SLI(reconcile rate, workqueue depth, REST client errors)를 수집하고, 선언적 policy로 CI 게이트 판정을 내리는 **shift-left operational SLI guardrail 라이브러리**다.

- correctness 테스트 도구가 아님 (기능이 맞는지 = E2E의 역할)
- 프로덕션 모니터링 시스템이 아님 (point-in-time guardrail)
- operator 코드를 수정하지 않음 (/metrics를 외부에서 scrape)
- 측정 실패는 테스트 실패가 아님 (non-fatal, safety-first)

**최종 목표**: 한국 오픈소스 공모전 제출 및 1등.

---

## 현재 상태 (2026-05-11 기준)

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
- [ ] `v0.1.0` git tag + GitHub release

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

internal/gate            ← policy 평가 (threshold/regression/reliability)
cmd/slint-gate           ← CLI entrypoint
.github/actions/slint-gate ← GitHub Composite Action
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
fail_on:
  - "threshold_miss"
  - "regression_detected"
```

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
