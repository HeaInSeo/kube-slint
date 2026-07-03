# Changelog

모든 변경사항은 이 파일에 기록됩니다.
형식은 [Keep a Changelog](https://keepachangelog.com/ko/1.1.0/)를 따릅니다.

## [Unreleased]

### Changed

- `internal/gate/gate.go`: `reliability.collectionStatus == "Failed"`는 `reliability.required` 설정과 무관하게 무조건 `NO_GRADE`로 승격됨 (기존에는 threshold 규칙이 없고 `reliability.required: false`이면 조용히 `PASS`가 나올 수 있었음). 새 reason 코드 `COLLECTION_FAILED` 추가.
- `internal/gate/gate.go`: regression 검사가 metric 방향(threshold rule의 `operator`)을 인식함 — `<=`/`<`는 lower-is-better, `>=`/`>`는 higher-is-better로 취급하여 개선(improvement)을 더 이상 회귀로 오탐하지 않음. 방향을 알 수 없는 연산자(`==` 등)는 기존 대칭 tolerance 검사를 유지.
- `pkg/slint/session.go`, `pkg/slint/fetcher_curlpod.go`: curl-pod 기반 fetch(`PreFetch`/`Fetch`)의 외부 context timeout이 더 이상 `ScrapeTimeout`(2분)으로 `WaitPodDoneTimeout`(5분)+`LogsTimeout`을 무효화하지 않음 — `WaitPodDoneTimeout+LogsTimeout+여유`로 계산.
- `pkg/slint/sweep.go`: orphan sweep 제외 셀렉터(`slint-run-id!=...`)가 다른 셀렉터들과 동일하게 `SanitizeKubernetesLabelValue`를 거침.
- `pkg/slint/attach.go`: `SessionConfig.Token`이 비어 있어도 더 이상 테스트가 실패하지 않음 — 기본 curlpod fetcher는 pod에 마운트된 ServiceAccount 토큰을 사용하므로 `Token` 필드는 커스텀 Fetcher를 위한 호환성 필드로만 남음.
- `pkg/slo/fetch/curlpod/client.go`: 생성되는 curl pod PodSpec에 `automountServiceAccountToken: true`를 명시 — ServiceAccount 기본값에 의존하지 않음.
- `cmd/slint-gate/diagnose.go`: `POLICY_INVALID` 진단 힌트에 `schema_version`/`fail_on`/`reliability.min_level` 원인을 명시 (기존에는 YAML 문법과 operator만 언급해 원인을 못 찾기 쉬웠음).
- `examples/kind-hello-operator/manifests/rbac.yaml`: `ClusterRole`/`ClusterRoleBinding` → 네임스페이스 스코프 `Role`/`RoleBinding`으로 변경 (`slint-gate init --emit-rbac` 템플릿과 정합).
- `pkg/slo/fetch/promtext`: bare-name 메트릭 합산 로직(`Aggregate`/`ParseTextToMapWithAggregates`)을 curlpod fetcher 전용 코드에서 공용 패키지로 이동하여 curlpod/portforward fetcher가 동일한 metric 의미를 갖도록 통일. 실제 unlabeled series가 있으면 덮어쓰지 않고, histogram bucket(`le` 레이블)/summary quantile(`quantile` 레이블)은 합산 대상에서 제외하도록 개선.

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
