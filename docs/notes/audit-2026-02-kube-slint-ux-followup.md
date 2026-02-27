# Audit Follow-up: UX & Docs (Feb 2026)

## 1. 감사 결론 요약 (Audit Conclusion)
* **방향 정렬(Aligned)**: 운영 코드 침투 금지, E2E Hook 기반, 에러 격리(Best-effort)라는 `kube-slint` 디자인 철학이 구현과 명확히 잘 일치합니다.
* **우선 순위 판별**: 장기적 구조 논의 혹은 문서 개편보다 하네스의 엣지 케이스 안정성 달성이 시급하여 `[패치 T-2]` 를 먼저 진행했습니다.

## 2. [패치 T-2] 작업 결과
본 패치에서 완료된 하네스 신뢰도 보강 작업은 아래와 같습니다:
* **Cleanup mode matrix 보강** (`session_test.go`): `always`, `on-success`, `manual` 등 cleanup 조합 모드에 따른 분기 오판/누락 방지 테스트 신설 완료.
* **CheckGating 동작 보강** (`propagation_test.go`): `GateOnLevel` 정책(warn/fail)과 `SLI Status`의 조합에 따른 정책 결정 신뢰도 테스트 신설 완료.
* **Preset/default Specs Smoke** (`presets_test.go`): 기본 제공되는 v3 Specs가 panic/error 없이 로드되고 최소 요건(`ID`, `Inputs`, `Judge`)을 갖추었음을 스모크 레벨에서 검증.

## 3. 향후 후속 과제 (Post T-2)
이후 문서화와 UX 사용성 강화를 위해 다음 항목들을 별도 과제로 진행해야 합니다.

### A. 결과물(sli-summary.json) 해석 가이드라인 작성
* `ConfidenceScore`의 단계별(1.0, 0.8, 0.0) 해석 기준
* `SLI Status`의 보류 상태(`Partial`, `Skip`) 발생 시 실 사용자가 어떻게 판단해야 하는지 판단표 제공.

### B. Custom SLI 튜토리얼 신설
* 개발자 통합 가이드 문서(`docs/current/kube-slint 개발자 통합 가이드 v1.2`)에, `Attach()` 안에서 `SessionConfig.Specs` 슬라이스를 구축하여 도메인 특화 메트릭 평가를 연동하는 50줄 이내의 예시 코드 추가.

### C. IO 예외(Artifact 저장 실패)에 대한 주의사항 복원 방안
* 과거 문서의 "Do not assume artifact exists when err == nil" 원칙이 `v4.4` 문서에서 누락된 점을 확인. `Strictness` 설명 파트에 오류 없는(`nil`) 결과 반환에도 실제 파일 쓰기는 실패할 수 있음을 경고.
