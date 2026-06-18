package sync

import "encoding/json"

// JSONOutput is the structured output for --output=json mode.
type JSONOutput struct {
	Catalog   string       `json:"catalog"`
	CatalogID string       `json:"catalog_id"`
	Counts    Counts       `json:"counts"`
	Changes   []JSONChange `json:"changes"`
}

// JSONChange is a single change entry in JSON output.
type JSONChange struct {
	Op         string               `json:"op"`
	ExternalID string               `json:"external_id"`
	Name       string               `json:"name"`
	FieldDiffs map[string][2]string `json:"field_diffs,omitempty"`
}

// PlanToJSON serializes a Plan as indented JSON bytes.
func PlanToJSON(plan *Plan) ([]byte, error) {
	out := JSONOutput{
		Catalog:   plan.Catalog,
		CatalogID: plan.CatalogID,
		Counts:    plan.Counts,
	}
	for _, c := range plan.Changes {
		out.Changes = append(out.Changes, JSONChange{
			Op:         c.Op,
			ExternalID: c.ExternalID,
			Name:       EntityName(c),
			FieldDiffs: c.FieldDiffs,
		})
	}
	return json.MarshalIndent(out, "", "  ")
}
