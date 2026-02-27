# kube-slint Project Progress Log

This file tracks the incremental stages of kube-slint work.
Update this file at the **start and end** of every stage/task.

---

## Current Status: Stage (Completed) — Cleanup Audit & Diagnostics

**Branch:** `main`
**Last updated:** 2026-02-27

### Current Focus

* 저장소 파일/구조/테스트 신뢰성에 대한 진단(Audit) 완료
* 증거 기반 정리(Cleanup) 대상 파악 및 우선순위가 제안된 보고서 작성 완료

### Definition of Done (DoD)

* [x] 저장소 파일 정리 필요성 파악 (obsolete/legacy/keep 등)
* [x] 코드 구조 및 책임 분리 진단
* [x] 테스트 신뢰도 분석 및 구멍 발견
* [x] 정리 계획이 담긴 리포트(`docs/notes/cleanup-audit-report-2026-02-27.md`) 생성

### Next command to run

* (User 리뷰 후 결정, 현재 없음)

### If blocked, fallback check

* (해당 없음)

---

## Completed Items

### Stage 7 — Implementation & Stabilization

* 기초 하네스 구현 및 안정화 완료
* GitHub Actions lint/test 통과 상태 확보 완료

### Stage T-2 — Harness Test Reinforcement 2nd

* Cleanup mode matrix 테스트 보강 완료
* CheckGating 테스트 보강 완료
* preset/default specs smoke 테스트 보강 완료

### Stage Audit & UX/Docs Reinforcement (Post-T-2)

* (Audit) 계측 실패 격리, E2E Hook 기반 등의 핵심 철학 정렬 확인 완료
* (Docs) `sli-summary.json` 결과 해석 가이드 보강 완료
* (Docs) Custom SLI 튜토리얼(`SessionConfig.Specs`) 안내 완료
* (Docs) Artifact 존재 가정 금지(IO 실패 격리) 경고 문서화 완료
* (Docs) 초보자 가독성을 위한 상태 계층(Status Layers) 표 도입 및 JSON 예시 추가 조치 완료
* (Docs) 마감용 리터치를 통한 7.3/7.4 상태 표현 계층 및 JSON 해석 문장의 용어 정밀화 완료

### Stage E2E Final Verification

* (Verification) `test/e2e` 매니저 컨트롤러 구버전 테스트 코드 발견 및 무시(repository가 library로 전환된 철학에 맞지 않음). `test/e2e/harness`의 시뮬레이터 및 Go JSON 정합성 테스트로 Fallback 수행.
* (Verification) Gating/Strictness 실패 시 `harness.Attach` 에러 전파 흡수 여부 확인(테스트 실패시키지 않고 GinkgoWriter에 로그 남김 -> "테스트!=측정실패" 철학 준수).
* (Docs Patch) 섹션 6.3에 `Attach` 훅의 로그-only 에러 삼킴 규칙을 소규모 명시 패치하여 Artifact 부재 경고 타당성 최종 확인.

### Stage Phase A/B (T-3 SanitizeFilename 보강)

* (Phase A) 문서 v1.2 가이드 7.4항 "Partial" 조건 설명 시 평가 스킵이 아닌 보조 지표 누락 가능성을 명확히 분리 서술.
* (Phase A) PROGRESS_LOG 릴리즈 항목 중복 제거 및 구버전 (Current) 꼬리표 정리 완료.
* (Phase B) `test/e2e/harness/sanitize_test.go` 파일 구축. 빈 문자열(`""` -> `"unknown"`), 공백정리(`"  "` -> `"unknown"`), 경로구분자, 특수문자 치환 등 파일시스템 보호를 위한 10종 엣지케이스 Table-driven 테스트로 방어력 증명 완료 (기존 함수 수정 없이 통과).

### Stage Cleanup Audit & Diagnostics

* 저장소 구조/테스트 신뢰성에 대한 진단(Audit) 실시 및 `docs/notes/cleanup-audit-report-2026-02-27.md` 제출.
* 발견 사항 요약: 루트 디렉토리의 임시 로그(`.log`, `e2e.test`) 방치, `test/e2e` 폴더 내의 Dummy Controller 배포 코드가 더 이상 유효하지 않은 Legacy 상태(Broken E2E), `pkg/kubeutil`의 YAML Sprintf 하드코딩 부채(`TODO(security)`), 그리고 `test/e2e/harness/session.go` 내의 Fetcher Adapter 결합 관찰.

---

## Pending Items

### Next Stage (planned)

* [ ] Release & Tagging (릴리즈 준비)
* 목적: Final Verification 정합성 점검이 통과되었으므로 현재 저장소 꼴을 기반으로 버전을 커팅할 준비.

### Proposed Next Stage (pending approval)

* [ ] Cleanup Execution (Phase 1, 2, 3)
* 제안 내역:
  1. (Quick Win) 루트 디렉토리의 의미 없는 아티팩트(`*.log`, `*.test`, `cover.out`, `TODO.md` 등) 정리
  2. (Structure) 결합도를 높이는 `session.go`에서 `curlPodFetcher` 모듈 분리 및 `kubeutil` 내 `TODO(security)`/`TODO(refactor)` 정리
  3. (Testing) `kube-slint` 라이브러리에 적합하지 않게 망가져버린 순수 통신망 `test/e2e` 재편성
* 이 중에서 어느 Phase부터 시작할지 논의 및 결정 필요.
* 승인 필요: **Yes (user + ChatGPT)**

### Follow-up (deferred)

* [ ] `sli-summary.json` CLI Console Output 요약 기능 지원

### Backlog (optional)

* [ ] Trigger-based 경계 지원 (Annotation/Condition 기반)

---

## Recent Validation Baseline

* `golangci-lint ./...` — PASS (2026-02-27)
* `go test ./test/e2e/harness/...` — PASS (2026-02-27, harness scope)
* `go mod tidy` — PASS (2026-02-27)
* `git diff --exit-code go.mod go.sum` — PASS (2026-02-27)

---

## Working Guardrails (Do not regress)

* Non-invasive instrumentation (no production operator code instrumentation changes)
* E2E Hook-based measurement
* Measurement failure != test failure (best-effort / warn / skip)
* Raw metrics (`/metrics`) vs summarized output (`sli-summary.json`) separation
* Keep scope small; defer instead of expanding

---

## Deferred / Risks (rolling)

* E2E Hook 내부 에러가 외부 라이브러리(Ginkgo 등)에 전파될 때 환경마다 exit code나 Fail() 처리가 상이할 수 있는 리스크

---

## Notes for Next Agent / Next Chat (short)

* Start by reading `docs/PROGRESS_LOG.md`.
* Confirm Current Status + DoD before editing.
* Record out-of-scope findings in Deferred first.
