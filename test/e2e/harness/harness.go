// Package harness preserves the historical test/e2e import path.
//
// New consumers should import github.com/HeaInSeo/kube-slint/pkg/slint.
package harness

import (
	"github.com/HeaInSeo/kube-slint/pkg/slint"
	"github.com/HeaInSeo/kube-slint/pkg/slo/spec"
)

type SessionConfig = slint.SessionConfig
type Session = slint.Session
type DiscoveredConfig = slint.DiscoveredConfig
type ConfigSource = slint.ConfigSource

type OrphanSweepOptions = slint.OrphanSweepOptions
type SweepRequest = slint.SweepRequest
type SweepApply = slint.SweepApply
type SweepSummary = slint.SweepSummary
type SweepItem = slint.SweepItem
type SweepResult = slint.SweepResult

var DevSweepOptions = slint.DevSweepOptions
var CISweepOptions = slint.CISweepOptions

var NewSession = slint.NewSession
var Attach = slint.Attach
var DiscoverConfig = slint.DiscoverConfig
var SanitizeFilename = slint.SanitizeFilename
var CheckStrictness = slint.CheckStrictness
var CheckGating = slint.CheckGating
var WriteSweepResultJSON = slint.WriteSweepResultJSON

func DefaultV3Specs() []spec.SLISpec {
	return slint.DefaultV3Specs()
}

func BaselineV3Specs() []spec.SLISpec {
	return slint.BaselineV3Specs()
}

func DefaultSpecs() []spec.SLISpec {
	return slint.DefaultSpecs()
}

func BaselineSpecs() []spec.SLISpec {
	return slint.BaselineSpecs()
}
