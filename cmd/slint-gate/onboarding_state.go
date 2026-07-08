package main

import (
	"strings"

	"github.com/HeaInSeo/kube-slint/pkg/gate"
	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
)

// onboardingState is the shared "where is this project in the onboarding
// loop" snapshot used by both quickstart (prints one suggested command) and
// wizard (prompts for input and runs the step itself). Keeping the
// detection logic in one place means the two commands can never disagree
// about what state the project is in.
type onboardingState struct {
	HasPolicy bool

	HasSummary     bool
	SummaryInvalid bool // file exists but failed to load (distinct from never-produced)
	SummaryLoadErr error
	Summary        summary.Summary

	GateResult string // only set when HasPolicy && HasSummary

	BaselineRequested bool // --baseline was given at all
	HasBaseline       bool
}

// inspectOnboardingState reads the policy/summary/baseline files (if
// present) and evaluates the gate if enough is available to do so. It never
// fails — a missing or invalid file is reflected in the returned state, not
// returned as an error, since "nothing is set up yet" is an expected state
// for both quickstart and wizard to handle, not a fatal condition.
func inspectOnboardingState(policyPath, summaryPath, baselinePath string) onboardingState {
	hasPolicy := fileExists(policyPath)

	summaryFileExists := fileExists(summaryPath)
	s, summaryErr := summary.LoadFile(summaryPath)
	// A summary that exists but fails to load is a different problem
	// ("it's broken, go inspect it") than one that was never produced yet
	// ("go run your E2E test") -- both are HasSummary=false, but callers
	// need to tell them apart.
	summaryInvalid := summaryFileExists && summaryErr != nil
	hasSummary := summaryErr == nil

	var gateResult string
	if hasPolicy && hasSummary {
		result := gate.Evaluate(gate.Request{
			MeasurementPath: summaryPath,
			PolicyPath:      policyPath,
		})
		gateResult = result.GateResult
	}

	baselineRequested := strings.TrimSpace(baselinePath) != ""
	hasBaseline := baselineRequested && fileExists(baselinePath)

	return onboardingState{
		HasPolicy:         hasPolicy,
		HasSummary:        hasSummary,
		SummaryInvalid:    summaryInvalid,
		SummaryLoadErr:    summaryErr,
		Summary:           s,
		GateResult:        gateResult,
		BaselineRequested: baselineRequested,
		HasBaseline:       hasBaseline,
	}
}

// onboardingStep identifies which onboarding action should happen next,
// following the precedence: fix a broken summary; get a policy in place;
// get a measurement in place; make the gate PASS; approve a baseline; wire
// CI. quickstart formats this into a suggested command string; wizard acts
// on it directly.
type onboardingStep int

const (
	stepInspectInvalidSummary onboardingStep = iota
	stepInit
	stepRecommendPolicy
	stepRunE2E
	stepInspectFailedGate
	stepApproveBaseline
	stepWireCI
)

func nextOnboardingStep(st onboardingState) onboardingStep {
	switch {
	case st.SummaryInvalid:
		return stepInspectInvalidSummary
	case !st.HasPolicy && !st.HasSummary:
		return stepInit
	case !st.HasPolicy:
		return stepRecommendPolicy
	case !st.HasSummary:
		return stepRunE2E
	case st.GateResult != gate.GatePass:
		return stepInspectFailedGate
	case !st.HasBaseline:
		return stepApproveBaseline
	default:
		return stepWireCI
	}
}
