# Changelog

모든 변경사항은 이 파일에 기록됩니다.
형식은 [Keep a Changelog](https://keepachangelog.com/ko/1.1.0/)를 따릅니다.

## [Unreleased]

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
