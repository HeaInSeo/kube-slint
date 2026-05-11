# Changelog

모든 변경사항은 이 파일에 기록됩니다.
형식은 [Keep a Changelog](https://keepachangelog.com/ko/1.1.0/)를 따릅니다.

## [Unreleased]

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

[Unreleased]: https://github.com/HeaInSeo/kube-slint/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/HeaInSeo/kube-slint/releases/tag/v0.1.0
