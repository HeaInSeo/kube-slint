# Global Cleanup Diagnostics Audit (Repo-wide)
**Date:** 2026-02-27
**Status:** Diagnostics Audit Complete (Read-only)

## 1. Executive Summary

- **전역 정리 필요성 수준(Urgency)**: **Medium** (크리티컬 버그는 없으나 혼동을 주는 레거시와 부채가 산재되어 있음)
- **가장 큰 혼동/리스크 3개**:
  1. `presets/` 디렉토리 하위의 완전 주석 처리된(Commented-out) 레거시 파일들. 현재 하네스 철학과 어긋남.
  2. `test/e2e` 기본 통합 범위에서 격리된 더미 컨트롤러 기반 옛날 테스트. (실제 통합 환경 보호의 부재)
  3. `scripts/check-slo-metrics.sh` 등 라이브러리 형태의 저장소에 어울리지 않는 과거 Bash 스크립트 산재.
- **추천 방향**: 구버전 아티팩트들은 과감히 `Likely obsolete` 처리하여 저장소를 경량화(Library-centric)하고, 테스트 자산 복구를 후속 필수 과제로 지정.

---

## 2. Scope & Method

- **범위**: `kube-slint` 저장소 루트부터 하위 모든 디렉토리 전수 검사 (특히 `presets/`, `scripts/`, `docs/`, `pkg/`, `test/`)
- **분석 방법론**: 파일명 및 `grep_search` (`// func`, `TODO`, 등) 명령어 기반으로 코드 잔재와 구조적 결합도, 테스트 자산을 진단함.
- **제약 사항**: 어떠한 파일도 임의로 수정/삭제하지 않은 Read-only 관찰 보고서임.

---

## 3. Findings — Global Inventory of Cleanup Candidates

| 디렉토리 / 파일 | 분류 라벨 | 근거 (왜 남아있을까 / 문제점) | 권장 조치 |
| --- | --- | --- | --- |
| `presets/` (`controller_runtime`, `my_operator`) | `Likely obsolete` 또는 `Revive candidate` | (과거 Registry 방식 잔재) 파일 내 함수 전체가 주석 처리됨. | 코드로 남길 이유가 없음. `docs/examples/` 하위에 JSON 예시로 현대화하거나, 완전 삭제 권장. |
| `scripts/check-slo-metrics.sh` | `Needs confirmation` | (이전 수동 curl 확인용) v4 하네스 도입 이전에 쓰이던 스크립트로 추정됨. | 현 하네스와 겹치는 스코프이므로 삭제(Obsolete) 고려. |
| `docs/old/` 하위 전체 폴더/파일 | `Keep (historical reference)` | 기획 당시의 고민(v1~v4 설계안)이 보존되어 아카이빙 목적으로 잔존. | 유저는 `docs/current/`만 볼 수 있도록 리드미에 명시. |
| `pkg/kubeutil` 내 `TODO(security)` / `TODO(refactor)` | `Keep but clarify/refactor` | 과거 Standalone 시절 테스트 지원용 임시 로직들(Wait/RBAC Sprintf). 당장 크래시를 유발하지 않아서 보존. | 이후 구조 개편 페이즈(Phase 4)에서 리팩터링 우선순위를 높여 해소. |
| `test/e2e/...` (`manifests/` 등) | `Keep (needs rewrite)` | 현재 `legacy_e2e` 로 격리되어 있지만, Library 관점의 테스트 자산으로 변모가 필요. | 실제 Library 소비자 관점의 E2E 테스트로 현대화(Revive/Rewrite). |

---

## 4. Findings — Commented-out / Legacy / Placeholder Code

1. **`presets/controller_runtime/v1.go` 및 `presets/my_operator/v1.go`**
   - **상태**: 100% (모든 함수가) `// func ...` 형태로 주석 처리되어 있습니다.
   - **이유**: 과거 "코드 레벨에서 Preset Registry를 만들고 동적으로 바인딩한다"는 구상 하에 만들어진 Draft였으나, "불침투적 외부 JSON Spec 주입" 방식으로 진화하면서 버려진 코드입니다.
   - **조치 제안**: 삭제. 코드로 놔둘 의미가 없으며, JSON/YAML Spec 형태의 Example 문서로 전환하는 것이 하네스 철학에 부합합니다.

2. **`test/e2e/e2e_test.go` 내부 주석 블록들**
   - **상태**: `metricsFetcher` 할당부나 `// cmd = exec.Command("kubectl", "label"...` 코드들이 방치 주석화 되어있습니다.
   - **이유**: 테스트가 깨진 상태에서 여러 번 디버깅을 거치다 포기(혹은 격리)하면서 찌꺼기가 남은 것으로 추정.
   - **조치 제안**: E2E 재편성 단계(재구축)에서 새 백지 상태로 덮어쓰며 자연스럽게 버려질 구간입니다.

---

## 5. Findings — Code Structure / Responsibility Boundaries

1. **`test/e2e/harness/session.go` 내의 Fetcher Adapter 결합 문제 (중위험)**
   - **상태**: `session.go` 파일 432 라인 부근에 `default Fetcher (curlPod)` 로직이 함께 박혀있습니다.
   - **문제**: 본래 하네스(Session)는 "어떻게 가져오느냐(Fetcher)"의 구체적 방법을 모르는 오케스트레이션 층이어야 합니다.
   - **조치 제안(작게 나눠서 가능)**: `curlPodFetcher` 구조체를 분리하여 `pkg/slo/fetch/curlpod`로 이동시키는 리팩토링 권장.

---

## 6. Findings — Test Reliability & Test Asset Cleanup

- **현재 테스트 보장 (Unit & Harness)**: `pkg/slo/`와 `test/e2e/harness/` 단위별 엣지케이스(Strictness, JSON 산출물 정확도)는 아주 건실합니다.
- **미보장 자산 (E2E)**: 레거시 `//go:build legacy_e2e`에 갇혀 버린 옛 테스트 자산. 프로젝트가 라이브러리로 방향을 틀었기 때문에 과거처럼 `make deploy`로 컨트롤러 띄우기를 시도하다 모두 망가집니다.
- **릴리즈 차단 여부**: `Non-blocking but high-value cleanup`. 핵심 라이브러리 기능이 고장난 것은 아니나, 오픈소스로 릴리즈할 때 "이 라이브러리를 붙인 오퍼레이터는 이렇게 개발된다"를 보여줄 E2E 샘플(가장 훌륭한 문서 역할)이 부재하게 된 리스크가 있습니다.

---

## 7. Cleanup Execution Plan Proposal (초안)

어떤 단계로 정리할 타당할지 제안하는 승인 대기용 단계 목록입니다.

* **Phase 1: Quick Kills (잔해/주석 삭제)**
  - 내용: `presets/` 디렉토리 완전 통째 삭제 및 `scripts/check-slo-metrics.sh` 폐기 검토.
  - 목적: 무의미한 저장소 용량과 혼동 요소 즉시 차단.
* **Phase 2: Extract & Structure (구조/결합도 해소)**
  - 내용: `session.go` 하단의 Fetcher 분리, `pkg/kubeutil` 부채 상환.
  - 목적: 라이브러리화 철학의 순수성 회복.
* **Phase 3: Modernize Test Assets (레거시 E2E 철거 및 재건)**
  - 내용: `legacy_e2e` 코드를 삭제하고, `test/e2e/example_operator/` 형태의 모조 오퍼레이터를 통해 "라이브러리 소비 관점"의 통합 테스트 신규 작성.
  - 목적: 회귀 방어력을 복구하고 동작하는 샘플 통합안 제시.

---

## 8. Decision Checklist (결정 요청 사항)

- [ ] (Phase 1 판단) `presets/` 통째로 날려도 괜찮은가? (JSON 샘플이 필요하다면 docs에 추가로 작성할지 여부)
- [ ] (Phase 2, 3 우선순위) 당장 릴리즈(v1.0.X)를 위해 어디까지의 청소(Cleanup)를 선제적으로 허용할 것인가?
