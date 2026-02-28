# kube-slint Project Progress Log

This file tracks the incremental stages of kube-slint work.
Update this file at the **start and end** of every stage/task.

---

## Current Status: Stage (Active) — Release & Tagging Preparation

**Branch:** `main`
**Last updated:** 2026-02-28

### Current Focus

* (Release Prep) `v1.0.0-rc.1` 릴리즈를 위한 최종 검증 및 문서화 준비.
* **검증 결과 (Baseline 재확인)**: 
  - Git tree clean (`git status --short`)
  - Go module integrity (`go mod tidy`, `git diff --exit-code`)
  - Harness Unit/Integration (`go test ./test/e2e/harness/...`) 모두 정상 통과.
* **문서 정합성**: `cleanup-policy-decision-input-2026-02-28.md` 문서를 "승인 대기"에서 "과거 실행 완료(Archive)" 톤으로 교정함.
* **릴리즈 노트 초안**: `docs/RELEASE_NOTES_DRAFT.md` 작성 완료.
* **태그 전략 수립**: `v1.0.0-rc.1` 제안 (사유명시, 실행 대기).

### Definition of Done (DoD)

* [x] 정책 문서 하단 노트 문구(미래형 → 아카이브 톤) 정리 완료
* [x] 릴리즈 준비 체크리스트 결과가 로그에 기록됨
* [x] 릴리즈 노트 초안 작성 완료
* [x] 태그 전략(권장 버전/근거/명령어) 정리 완료
* [x] `PROGRESS_LOG.md` 갱신 및 실제 태그 생성 미실행 준수

### Next command to run

* (사용자 보고 및 실제 Release / Tagging 단행 대기)

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

### Stage Cleanup Execution (Phases 1 & 3)

* (Phase 1) 루트 및 각종 디렉토리에 산재되어 있던 방치 파일(`TODO.md`, `code_review.md`, `test_full_v*.log`, `cover.out`, `e2e.test` 등) 삭제 및 Git Tracked 로그 파일(`lint.log` 등)을 `git rm` 명령으로 저장소 인덱스에서 정리함. 
* (Phase 3) Library화로 인해 동작하지 않는 `test/e2e` 하위 레거시 테스트(`e2e_test.go`, `e2e_suite_test.go`)들에 `//go:build legacy_e2e` 빌드 태그를 부여하여 표준 `go test ./...` 및 CI 범위에서 격리(Quarantine) 처리함. 
* (Phase 3) `test/e2e/README.md`를 생성하여 해당 E2E 테스트가 제외된 이력을 명시하고, 파일 경로를 정확히 `test/e2e/...` 하위로 정정함. `Makefile` `test` 커맨드는 `grep -v /e2e` 방식 대신 기본 동작으로 정상화.
### Stage Consistency Patch

* (Correction) 이전에 지워지지 않고 Git에 임시로 Tracked되어 남아 있던 `lint.log`, `test_full.log` 등 4개 파일을 `git rm`하여 증거 기반으로 제거함.
* (Correction) `test/e2e/README.md` 내에 기재된 `e2e_test.go`의 경로 누락(`test/e2e_test.go` -> `test/e2e/e2e_test.go`)을 실제 파일 시스템 구조와 맞게 정합성 수립.
* (Correction) `PROGRESS_LOG.md` 내의 "100%", "영구 제거", "Ready for Release"와 같은 과장 표현 및 릴리즈 독단 판정 문구를 모두 객관적("격리", "정리", "상태 갱신")인 표현으로 배제함.

### Stage Cleanup Execution Phase 1-lite & 3-prep (Policy First)

* 명백한 잔해로 판별된 최상위 `cover.out` 등의 물리적 흔적 삭제 불가 여부 확인 및 Gitignore(`bin/`) 통제 추가 조치 (Read-only 기조 유지).
* 애매한 항목 삭제 대신 처리 정책 결정을 위해 `docs/notes/cleanup-policy-decision-input-2026-02-28.md` 문서 도출 (`presets/`, `scripts/check-slo-metrics.sh` 정책 비교 및 삭제/이관 추천안). 과감한 삭제 전 사용자 결정 요쳥.
* 소비자 단위로써의 테스트를 재건하기 위한 아키텍처 초안(`docs/notes/e2e-modernization-prep-2026-02-28.md`) 수립, Mock Server 기반의 Harness Integration Test 전략 선제안 (대규모 삭제 전초 작업).

### Stage A — Policy Checkpoint gates (Stop-and-Report)

* 증거 확보 전 삭제 금지 기조에 따라, `cleanup-policy-decision-input-2026-02-28.md` 내에 기재된 조건부 삭제 조항(`Delete (Conditional)`)을 단순히 '문서 예제 존재 확인'에서 **'Phase 4-a 소비자 검증 자산 성공 확보'**라는 구체적이고 물리적인 Execution Gate로 치환함.
* `pkg/` 변경 금지 및 `test/consumer-onboarding/` 산출물 배치 준수 가이드라인 등을 공식화하여 문서 간 정합성을 일치시킴.
* Stage B 시작 시점에 정책 체크박스 문구를 정밀화("Approve conditional delete policy (JSON examples + Phase 4-a success evidence)?")하는 Preflight 반영 완료.

### Stage B — Phase 4-a: Consumer Onboarding Probe (Go import)

* `test/consumer-onboarding/kubebuilder-default-sli/` 하위에 최소화된 빈 깡통 Reconciler 기반 샘플 구축.
* `envtest`를 사용해 테스트 클러스터 메모리에 매니저를 띄우고 `kube-slint` Harness `NewSession` -> `Start()` -> `End()` 사이클 호출 확인 (PASS 증거 획득).
* **관찰 결과 (4분류 분석)**:
  1. **문서 UX 문제**: `harness` API 사용 시 필수 설정(`Namespace`, `MetricsServiceName` 등)이 무엇인지 컴파일러 레벨에서 직관적이지 않음 (추후 가이드라인 보강 필요 증거).
  2. **API/인터페이스 문제**: 소비자 입장에서 `spec.PromMetric()` 보다 `spec.UnsafePromKey()`를 써야 하는 등 Spec 선언 과정의 구조체가 모호함.
  3. **테스트 자산 배치/구조 문제**: `setup-envtest` 바이너리(`test-operator/bin/k8s`)가 상위 폴더에 의존하여 Consumer 측 복사(cp)가 필요했음 (단독 실행 배포 시 약점). 
  4. **로깅/디버깅 문제**: `Session.Start()` 실행 시 Endpoint 스크랩 실패 등은 `kube-slint [discovery]:` 등 유의미한 표준 출력 정보가 다수 발생하어 쉘 스크립트(`check-slo-metrics.sh`) 없이도 로깅 수준이 충분함을 교차 검증함.

### Stage C — 정책 삭제 조건 재평가 (Evidence-based Judgment)

* Stage B의 결과를 파악하여 `cleanup-policy-decision-input-2026-02-28.md`의 조건부 삭제 조항 달성 여부를 판정함 (물리 삭제 절대 금지 원칙 준수).
* **`presets/` 판정**: Stage B 통합 테스트에서 패키지 없이 순수 JSON-string 형태로 정상 작동함을 증명. 조건 충족(Condition Met).
* **`scripts/check-slo-metrics.sh` 판정**: Stage B 구동 시 자동화된 파이프라인(Session Engine)이 뿜어내는 수많은 scrape 에러/로그가 디버깅에 충분하다고 판단됨. Phase 4-a / 4-b 필수 조건은 OR 조건(하나면 충분)으로 해석됨. 조건 충족(Condition Met via Phase 4-a).
* **Stage D와의 연결**: Stage B는 "라이브러리를 임포트하는 Go 소비자"의 입장을 대변함. 쿠버네티스 환경에 인프라(Kustomize Base/Overlays 등)를 심는 "운영/배포 소비자"의 입장은 별개의 검증이 필요함. 따라서 `check-slo-metrics.sh`의 삭제 근거는 확보되었으나, Kustomize Consumer UX를 다루는 Stage D(Phase 4-b)는 인프라 프로비저닝 구조 정합성 확인을 위해 독립적으로 수행되어야 함.

### Stage D — Phase 4-b: Kustomize Consumer UX Probe (Remote Resource)

* Kustomize 환경에서 `kube-slint` 인프라를 소비하는 외부 오퍼레이터의 UX 검증을 위해 `test/consumer-onboarding/kustomize-remote-consumer` 자산을 구축.
* 테스트 경로: `github.com/HeaInSeo/kube-slint//config/default?ref=0f48f...` 및 `//config/samples/prometheus?ref=0f48f...`
* **관찰 결과 (4분류 분석)**:
  1. **문서 UX 문제**: `README.md`는 Remote 핀 고정의 중요성을 잘 명시하나, `config/default`가 빈 껍데기임을 은연중에 인정하며 "로컬 복사 후 변형"을 권유함. 이는 원격 Kustomize 수입을 사실상 사용 불능하게 만드는 모순된 지시사항임.
  2. **Kustomize 경로/참조(ref pinning) 문제**: 문법적인 Kustomize Remote Fetch(`//`와 `?ref=`)는 정상 동작함. 툴링/경로상의 블로커는 없었음.
  3. **배치/구조 문제**: Standalone 파편이 남아있어, 실 사용(`config/samples/prometheus`) 시 리소스의 `matchLabels`가 라이브러리를 쓰는 타겟 Operator가 아니라 `kube-slint` 이름으로 하드코딩되어 있음. 유동적인 `nameReference`나 변수화 없이 Remote 가져오기는 불가능함(오류 없는 사일런트 실패 유발).
  4. **오류 메시지/디버깅 UX 문제**: Kustomize 빌드-어플라이는 에러 없이 통과해버리기 때문에, 사용자는 왜 자기 Metrics가 수집되지 않는지 Kubernetes 내부를 한참 뜯어봐야 하는 심각한 로깅/침묵의 UX를 가짐.

### Stage E — Approved Cleanup Execution & Final Synthesis

* 사용자 승인(User Approval)에 따라 확보된 정책 판단을 바탕으로, `presets/` 전체 디렉토리와 `scripts/check-slo-metrics.sh`를 소스 코드 트랙에서 영구 삭제(git rm) 함.
* `docs/notes/cleanup-policy-decision-input-2026-02-28.md`를 갱신하여 Condition Met 상태를 Execution Completed 상태로 변경함.
* **UX 부채 분리 (Stage D 파생)**: Kustomize 배포용 리소스(config/samples 등)가 `main` 브랜치에 그대로 남아있어 Remote Kustomize 접근 시 하드코딩 오류를 범하는 현상은 여전히 남아있음. 이는 삭제와는 별개의 문제이므로 Kustomize UX 부채로 라벨링하여 배포 구조 정립 과제(Backlog)로 격리함.

### Release & Tagging Preparation

* **태그 전략 (Tag Strategy)**: 제안 버전 `v1.0.0-rc.1`
  * **근거**: 라이브러리 E2E Harness 코어 로직이 안정화되었고, 불필요한 레거시(Standalone 찌꺼기)가 모두 청소됨. Stage B(Go import) 검증은 통과했으나, Kustomize UX 개선 및 Phase 3(Mock E2E) 구현 등 Consumer 온보딩을 위한 비기능적 백로그가 남았으므로 정식 `v1.0.0` 이전에 Release Candidate 1 을 발행하는 것이 적절함.
  * **명령어 (실행 대기용)**:
    1. `git tag -a v1.0.0-rc.1 -m "Release v1.0.0-rc.1: Cleanup and Harness Stabilization"`
    2. `git push origin v1.0.0-rc.1`
* **릴리즈 노트 초안**: `docs/RELEASE_NOTES_DRAFT.md` 참조.

---

## Pending Items

### Next Stage (planned)

* [ ] Release & Tagging Execution (실제 커팅 실행)
* 목적: 준비된 태그 전략에 맞추어 실제 버전을 퍼블리시하고 GitHub Release 페이지를 구성함.

### Proposed Next Stage (pending approval)

* [ ] **Phase 3 실제 구현 (Legacy E2E 대체)** 승인 대기.
  - 실행 계획: `e2e-modernization-prep` 문서를 바탕으로 Mock Operator 및 Harness 동작 테스트를 구축.
* [ ] (Optional) Kustomize Parameterization 구조 개편 착수.
* 승인 필요: **Yes (user + ChatGPT)**

### Follow-up (deferred)

* [ ] (Phase 2) `session.go`에서 `curlPodFetcher` 모듈 분리 및 `kubeutil` 내 `TODO(security)`/`TODO(refactor)` 해소 (릴리즈 이후로 지연)
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
