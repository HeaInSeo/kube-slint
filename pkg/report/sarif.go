package report

// Minimal, hand-rolled SARIF 2.1.0 envelope — only the fields kube-slint
// actually populates. No external SARIF dependency; validated in tests by
// unmarshaling into a small local struct covering the required subset.

const (
	sarifVersion = "2.1.0"
	sarifSchema  = "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json"
)

type sarifLog struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name    string      `json:"name"`
	Version string      `json:"version,omitempty"`
	Rules   []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string       `json:"id"`
	ShortDescription sarifMessage `json:"shortDescription"`
	FullDescription  sarifMessage `json:"fullDescription,omitempty"`
	HelpURI          string       `json:"helpUri,omitempty"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"` // "error" | "warning"
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
	LogicalLocations []sarifLogicalLoc     `json:"logicalLocations,omitempty"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifLogicalLoc struct {
	FullyQualifiedName string `json:"fullyQualifiedName"`
}

func toSARIF(r *Report) sarifLog {
	rules := make([]sarifRule, 0, len(r.Rules))
	for _, rule := range r.Rules {
		rules = append(rules, sarifRule{
			ID:               rule.ID,
			ShortDescription: sarifMessage{Text: rule.Title},
			FullDescription:  sarifMessage{Text: rule.Description},
			HelpURI:          rule.HelpURI,
		})
	}

	results := make([]sarifResult, 0, len(r.Findings))
	for _, f := range r.Findings {
		results = append(results, sarifResult{
			RuleID:  f.RuleID,
			Level:   string(f.Severity),
			Message: sarifMessage{Text: f.Message},
			Locations: []sarifLocation{
				{
					PhysicalLocation: sarifPhysicalLocation{
						ArtifactLocation: sarifArtifactLocation{URI: f.Location.File},
					},
					LogicalLocations: []sarifLogicalLoc{
						{FullyQualifiedName: f.Location.FullyQualifiedName()},
					},
				},
			},
		})
	}

	return sarifLog{
		Version: sarifVersion,
		Schema:  sarifSchema,
		Runs: []sarifRun{
			{
				Tool: sarifTool{
					Driver: sarifDriver{
						Name:    r.Tool,
						Version: r.ToolVersion,
						Rules:   rules,
					},
				},
				Results: results,
			},
		},
	}
}

// WriteSARIF writes r as a SARIF 2.1.0 log to path.
func WriteSARIF(path string, r *Report) error {
	return WriteJSON(path, toSARIF(r))
}
