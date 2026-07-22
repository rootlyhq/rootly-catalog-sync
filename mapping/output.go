package mapping

import (
	"fmt"

	"github.com/rootlyhq/rootly-catalog-sync/catalog"
	"github.com/rootlyhq/rootly-catalog-sync/config"
	"github.com/rootlyhq/rootly-catalog-sync/source"
	tmpl "github.com/rootlyhq/rootly-catalog-sync/tmpl"
)

func MapEntries(entries []source.Entry, out config.Output) ([]catalog.DesiredEntity, error) {
	result := make([]catalog.DesiredEntity, 0, len(entries))

	for i, entry := range entries {
		externalID, err := tmpl.Eval(out.ExternalID, entry)
		if err != nil {
			return nil, fmt.Errorf("entry %d: evaluating external_id: %w", i, err)
		}
		if externalID == "" {
			return nil, fmt.Errorf("entry %d: external_id evaluated to empty", i)
		}

		name, err := tmpl.Eval(out.Name, entry)
		if err != nil {
			return nil, fmt.Errorf("entry %d: evaluating name: %w", i, err)
		}
		if name == "" {
			return nil, fmt.Errorf("entry %d: name evaluated to empty", i)
		}

		var backstageID string
		if out.BackstageID != "" {
			backstageID, _ = tmpl.Eval(out.BackstageID, entry)
		}

		fields := make(map[string]string, len(out.Fields))
		for slug, fv := range out.Fields {
			val, err := tmpl.Eval(fv.Value, entry)
			if err != nil {
				return nil, fmt.Errorf("entry %d: evaluating field %q: %w", i+1, slug, err)
			}
			fields[slug] = val
		}

		result = append(result, catalog.DesiredEntity{
			ExternalID:  externalID,
			Name:        name,
			BackstageID: backstageID,
			Fields:      fields,
		})
	}

	return result, nil
}
