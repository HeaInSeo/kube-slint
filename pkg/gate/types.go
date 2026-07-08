package gate

// schemaVersion is the slint-gate-summary.json schema tag.
const schemaVersion = "slint.gate.v1"

// Gate result values.
const (
	// GatePass indicates all policy checks passed.
	GatePass = "PASS"
	// GateWarn indicates non-blocking warnings (e.g. first-run, reliability).
	GateWarn = "WARN"
	// GateFail indicates a policy violation that may fail CI.
	GateFail = "FAIL"
	// GateNoGrade indicates the gate could not be evaluated.
	GateNoGrade = "NO_GRADE"
)

const (
	evalEvaluated = "evaluated"
	evalPartial   = "partially_evaluated"
	evalNot       = "not_evaluated"
)

const (
	measOK                = "ok"
	measMissing           = "missing"
	measCorrupt           = "corrupt"
	measInsufficient      = "insufficient"
	measUnsupportedSchema = "unsupported_schema"
)

const (
	basePresent     = "present"
	baseAbsentFirst = "absent_first_run"
	baseUnavailable = "unavailable"
	baseCorrupt     = "corrupt"
)

const (
	policyOK      = "ok"
	policyMissing = measMissing
	policyInvalid = "invalid"
)

const (
	reasonThresholdMiss           = "THRESHOLD_MISS"
	reasonRegressionDetected      = "REGRESSION_DETECTED"
	reasonBaselineAbsentFirstRun  = "BASELINE_ABSENT_FIRST_RUN"
	reasonBaselineUnavailable     = "BASELINE_UNAVAILABLE"
	reasonBaselineCorrupt         = "BASELINE_CORRUPT"
	reasonMeasInputMissing        = "MEASUREMENT_INPUT_MISSING"
	reasonMeasInputCorrupt        = "MEASUREMENT_INPUT_CORRUPT"
	reasonMeasSchemaUnsupported   = "MEASUREMENT_SCHEMA_UNSUPPORTED"
	reasonResultStatusFail        = "RESULT_STATUS_FAIL"
	reasonPolicyMissing           = "POLICY_MISSING"
	reasonPolicyInvalid           = "POLICY_INVALID"
	reasonReliabilityInsufficient = "RELIABILITY_INSUFFICIENT"
	reasonCollectionFailed        = "COLLECTION_FAILED"
)

// Policy is the deserialized .slint/policy.yaml.
type Policy struct {
	SchemaVersion string          `yaml:"schema_version"`
	Thresholds    []ThresholdRule `yaml:"thresholds"`
	Regression    RegressionCfg   `yaml:"regression"`
	Reliability   ReliabilityCfg  `yaml:"reliability"`

	// FailOn is deprecated: use PromoteToFail instead. Both are honored
	// (union of the two) during the deprecation window; using FailOn
	// produces a PolicyWarnings entry recommending migration.
	FailOn []string `yaml:"fail_on"`

	// PromoteToFail lists which gate conditions ("threshold_miss",
	// "regression_detected") are promoted from WARN to FAIL. This is the
	// preferred name for what was previously FailOn — it disambiguates
	// from the CLI/action's --exit-on/exit-on, which controls process
	// exit code, not gate grade promotion.
	PromoteToFail []string `yaml:"promote_to_fail"`
}

// ThresholdRule is a single absolute threshold check.
type ThresholdRule struct {
	Name     string  `yaml:"name"`
	Metric   string  `yaml:"metric"`
	Operator string  `yaml:"operator"`
	Value    float64 `yaml:"value"`
}

// RegressionCfg holds regression comparison configuration.
type RegressionCfg struct {
	Enabled          bool    `yaml:"enabled"`
	TolerancePercent float64 `yaml:"tolerance_percent"`
}

// ReliabilityCfg holds reliability requirement configuration.
type ReliabilityCfg struct {
	Required bool   `yaml:"required"`
	MinLevel string `yaml:"min_level"`
}

// Check is a single policy check result.
type Check struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Status   string `json:"status"`
	Metric   string `json:"metric"`
	Observed any    `json:"observed"`
	Expected string `json:"expected"`
	Message  string `json:"message"`
}

// Summary is the slint-gate output schema (slint-gate-summary.json).
type Summary struct {
	SchemaVersion     string    `json:"schema_version"`
	GateResult        string    `json:"gate_result"`
	EvaluationStatus  string    `json:"evaluation_status"`
	MeasurementStatus string    `json:"measurement_status"`
	BaselineStatus    string    `json:"baseline_status"`
	PolicyStatus      string    `json:"policy_status"`
	Reasons           []string  `json:"reasons"`
	PolicyWarnings    []string  `json:"policy_warnings,omitempty"`
	EvaluatedAt       string    `json:"evaluated_at"`
	InputRefs         InputRefs `json:"input_refs"`
	Checks            []Check   `json:"checks"`
	OverallMessage    string    `json:"overall_message"`
}

// InputRefs records which files were used as inputs.
type InputRefs struct {
	MeasurementSummary string  `json:"measurement_summary"`
	PolicyFile         string  `json:"policy_file"`
	BaselineFile       *string `json:"baseline_file"`
}

// Request carries input paths for Evaluate.
type Request struct {
	MeasurementPath string
	PolicyPath      string
	BaselinePath    string // empty = first-run, no baseline
}
