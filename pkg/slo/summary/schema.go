// Package summary defines the sli-summary.json contract (schema slo.v3):
// the Summary/SLIResult shape every kube-slint measurement fetcher produces
// and every consumer (pkg/gate, cmd/slint-gate, pkg/slint) reads. A baseline
// file is not a distinct type — it is a plain Summary saved via WriteFile
// and reloaded via LoadFile, same as a regular measurement.
package summary

import (
	"fmt"
	"strings"
	"time"
)

// Measurement-contract versions (KSL-T5). The contract is an explicit fence:
//   - SchemaVersionLegacy ("slo.v3") is the historical contract. It carries no
//     comparability identity, so it can never be silently treated as trust-correct
//     protected evidence for baseline comparison.
//   - SchemaVersionTrust ("slo.v4") is the trust-correct contract. It adds the
//     per-SLI Comparability identity required for provable baseline comparability
//     (KSL-T4). A consumer that only understands the legacy contract must reject or
//     non-protect a v4 artifact rather than silently ignore its trust-required
//     fields.
//
// SchemaVersion remains the legacy version that current producers stamp; upgrading
// producers to emit the trust-correct contract with real comparability identity is
// owned by the later KSL-E packet, not KSL-T.
const (
	SchemaVersionLegacy = "slo.v3"
	SchemaVersionTrust  = "slo.v4"

	// SchemaVersion is the version current producers stamp (still the legacy
	// contract until KSL-E wires real trust-correct measurement).
	SchemaVersion = SchemaVersionLegacy
)

// supportedSchemaVersions is the set of measurement contracts a consumer accepts.
var supportedSchemaVersions = map[string]bool{
	SchemaVersionLegacy: true,
	SchemaVersionTrust:  true,
}

// ValidateSchemaVersion returns an error if s.SchemaVersion is empty or is not a
// supported measurement contract (legacy or trust-correct).
func ValidateSchemaVersion(s Summary) error {
	if s.SchemaVersion == "" {
		return fmt.Errorf("schemaVersion is empty")
	}
	if !supportedSchemaVersions[s.SchemaVersion] {
		return fmt.Errorf("unsupported schemaVersion %q (want %q or %q)",
			s.SchemaVersion, SchemaVersionLegacy, SchemaVersionTrust)
	}
	return nil
}

// IsTrustCorrectContract reports whether s uses the trust-correct measurement
// contract (slo.v4). Only a trust-correct measurement can carry the comparability
// identity required for a protected baseline comparison (KSL-T4/T5).
func IsTrustCorrectContract(s Summary) bool {
	return s.SchemaVersion == SchemaVersionTrust
}

// allowedStatuses is the enum of valid SLIResult.Status values.
var allowedStatuses = map[Status]bool{
	StatusPass:  true,
	StatusWarn:  true,
	StatusFail:  true,
	StatusBlock: true,
	StatusSkip:  true,
}

// Validate performs a comprehensive check of a Summary:
// schemaVersion must be supported, generatedAt must be non-zero, every
// SLIResult must have a non-empty and unique ID, and every SLIResult's
// Status must be one of the allowed enum values.
// External tools should call this after loading a summary to ensure contract compliance.
func Validate(s Summary) error {
	if err := ValidateSchemaVersion(s); err != nil {
		return err
	}
	if s.GeneratedAt.IsZero() {
		return fmt.Errorf("generatedAt is zero")
	}
	seenIDs := make(map[string]bool, len(s.Results))
	for i, r := range s.Results {
		if r.ID == "" {
			return fmt.Errorf("results[%d].id is empty", i)
		}
		if seenIDs[r.ID] {
			return fmt.Errorf("results[%d].id %q is a duplicate result ID", i, r.ID)
		}
		seenIDs[r.ID] = true
		if !allowedStatuses[r.Status] {
			return fmt.Errorf("results[%d].status %q is not a recognized status", i, r.Status)
		}
	}
	// A non-empty reliability status must be a recognized enum value. A mistyped
	// status (e.g. "Fialed") would otherwise slip past the collection-failure and
	// reliability checks and let a malformed artifact produce a protected grade.
	if s.Reliability != nil {
		if st := s.Reliability.CollectionStatus; st != "" && !isReliabilityStatus(st) {
			return fmt.Errorf("reliability.collectionStatus %q is not a recognized status", st)
		}
		if st := s.Reliability.EvaluationStatus; st != "" && !isReliabilityStatus(st) {
			return fmt.Errorf("reliability.evaluationStatus %q is not a recognized status", st)
		}
	}
	return nil
}

// isReliabilityStatus reports whether s is a recognized reliability status
// (Complete/Partial/Failed), case-insensitively — consumers compare these values
// case-insensitively, so validation must accept the same set.
func isReliabilityStatus(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "complete", "partial", "failed":
		return true
	default:
		return false
	}
}

// Status 는 SLIResult의 정규화된 평가 상태임.
type Status string

const (
	// StatusPass 는 성공을 나타냄.
	StatusPass Status = "pass"
	// StatusWarn 은 경고를 나타냄.
	StatusWarn Status = "warn"
	// StatusFail 은 실패를 나타냄.
	StatusFail Status = "fail"
	// StatusBlock 은 측정/파이프라인 실패를 나타냄.
	StatusBlock Status = "block"
	// StatusSkip 은 체크가 생략됨을 나타냄.
	StatusSkip Status = "skip"
)

// Summary 는 계약 출력임. 모든 측정 방식은 이 스키마로 수렴해야 함.
type Summary struct {
	SchemaVersion string    `json:"schemaVersion"`
	GeneratedAt   time.Time `json:"generatedAt"`

	Config RunConfig `json:"config"`

	Reliability *Reliability `json:"reliability,omitempty"`

	Results  []SLIResult `json:"results"`
	Warnings []string    `json:"warnings,omitempty"`
}

// Reliability 는 측정의 진단 및 신뢰도 상태를 포함함.
type Reliability struct {
	CollectionStatus string   `json:"collectionStatus,omitempty"` // Complete | Partial | Failed
	EvaluationStatus string   `json:"evaluationStatus,omitempty"` // Complete | Partial | Failed
	BlockedReason    string   `json:"blockedReason,omitempty"`
	MissingInputs    []string `json:"missingInputs,omitempty"`
	SkippedSLIs      []string `json:"skippedSLIs,omitempty"`

	ConfidenceScore *float64 `json:"confidenceScore,omitempty"` // 0.0 ~ 1.0 보조 지표

	ConfigSourceType string `json:"configSourceType,omitempty"` // injected | env | discovered
	ConfigSourcePath string `json:"configSourcePath,omitempty"`

	StartSkewMs     *int64 `json:"startSkewMs,omitempty"`
	EndSkewMs       *int64 `json:"endSkewMs,omitempty"`
	ScrapeLatencyMs *int64 `json:"scrapeLatencyMs,omitempty"`
}

// RunConfig 는 Summary에 포함됨 (분석 도구가 측정 방식에 구애받지 않도록 함).
type RunConfig struct {
	RunID      string            `json:"runId,omitempty"`
	StartedAt  time.Time         `json:"startedAt"`
	FinishedAt time.Time         `json:"finishedAt"`
	Mode       RunMode           `json:"mode"`
	Tags       map[string]string `json:"tags,omitempty"`
	Format     string            `json:"format,omitempty"`

	// EvidencePaths는 원시 아티팩트를 가리킴 (선택 사항).
	EvidencePaths map[string]string `json:"evidencePaths,omitempty"`
}

// RunMode 는 실행이 어떻게 수행되었는지를 설명함.
type RunMode struct {
	Location string `json:"location"` // "inside" | "outside"
	Trigger  string `json:"trigger"`  // "none" | "annotation"
}

// SLIResult 는 단일 SLI의 평가 결과를 포함함.
type SLIResult struct {
	ID          string `json:"id"`
	Title       string `json:"title,omitempty"`
	Unit        string `json:"unit,omitempty"`
	Kind        string `json:"kind,omitempty"`
	Description string `json:"description,omitempty"`

	// v3: 단일 수치 결과. 향후: p50/p99 등의 필드 추가 예정.
	Value  *float64           `json:"value,omitempty"`
	Fields map[string]float64 `json:"fields,omitempty"`

	Status Status `json:"status"` // "pass" | "warn" | "fail" | "block" | "skip"

	Reason string `json:"reason,omitempty"`

	InputsUsed    []string `json:"inputsUsed,omitempty"`
	InputsMissing []string `json:"inputsMissing,omitempty"`

	// Comparability is the trust-correct contract (slo.v4) identity that makes a
	// baseline comparison provably meaningful (KSL-T4). It is only meaningful under
	// the trust-correct measurement contract; a legacy (slo.v3) result never carries
	// it, so a legacy result can never satisfy protected comparability.
	Comparability *Comparability `json:"comparability,omitempty"`
}

// Comparability identifies every coordinate that affects the meaning of an SLI's
// value, so a regression comparison can be proven meaningful before it runs
// (KSL-T4). Two results are comparable only when all four coordinates are present
// and equal; window semantics come from an explicit WindowID, never inferred from
// run start/finish times.
type Comparability struct {
	// SLIContractID identifies the SLI/measurement contract semantics (what is
	// measured and how it is evaluated).
	SLIContractID string `json:"sliContractId,omitempty"`
	// SubjectID identifies the subject/context whose value is being measured.
	SubjectID string `json:"subjectId,omitempty"`
	// WindowID identifies the window/aggregation/query semantics.
	WindowID string `json:"windowId,omitempty"`
	// SourceConfigID identifies the source/config identity when it changes value meaning.
	SourceConfigID string `json:"sourceConfigId,omitempty"`
}

// Complete reports whether every comparability coordinate is present and
// non-blank. A whitespace-only coordinate is not a real identity, so it is
// treated as absent; an incomplete identity cannot prove comparability and
// yields NO_GRADE rather than a silent comparison.
func (c *Comparability) Complete() bool {
	return c != nil &&
		strings.TrimSpace(c.SLIContractID) != "" && strings.TrimSpace(c.SubjectID) != "" &&
		strings.TrimSpace(c.WindowID) != "" && strings.TrimSpace(c.SourceConfigID) != ""
}

// Equal reports whether two comparability identities match on every coordinate.
// A nil or incomplete identity is never equal to another.
func (c *Comparability) Equal(other *Comparability) bool {
	if !c.Complete() || !other.Complete() {
		return false
	}
	return c.SLIContractID == other.SLIContractID &&
		c.SubjectID == other.SubjectID &&
		c.WindowID == other.WindowID &&
		c.SourceConfigID == other.SourceConfigID
}

// ResultValues flattens Results into a map of SLI ID to value, omitting any
// result whose Value is nil (e.g. a skipped SLI). This is the single
// canonical way to go from a Summary to a comparable value map — gate
// evaluation, baseline diff, and baseline merge all need the same
// flattening and previously each reimplemented it separately.
func (s Summary) ResultValues() map[string]float64 {
	m := make(map[string]float64, len(s.Results))
	for _, r := range s.Results {
		if r.Value != nil {
			m[r.ID] = *r.Value
		}
	}
	return m
}

// EnsureFormat 은 schemaVersion을 보존하면서 포맷 힌트(기본값 v4)를 설정함.
func EnsureFormat(config map[string]any) map[string]any {
	if config == nil {
		config = map[string]any{}
	}
	if _, ok := config["format"]; !ok {
		config["format"] = "v4"
	}
	return config
}
