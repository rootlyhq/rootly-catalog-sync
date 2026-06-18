package sync

import (
	"fmt"

	"github.com/rootlyhq/rootly-catalog-sync/catalog"
)

// StaleChange describes a single plan change that no longer matches live state.
type StaleChange struct {
	ExternalID string
	Reason     string
}

// ValidatePlanFreshness checks whether a saved plan still matches live state.
// It returns a list of stale changes; an empty slice means the plan is fresh.
func ValidatePlanFreshness(plan *Plan, live []catalog.LiveEntity) []StaleChange {
	liveByExtID := make(map[string]catalog.LiveEntity, len(live))
	for _, e := range live {
		if e.ExternalID != "" {
			liveByExtID[e.ExternalID] = e
		}
	}

	var stale []StaleChange
	for _, c := range plan.Changes {
		switch c.Op {
		case OpCreate:
			if _, exists := liveByExtID[c.ExternalID]; exists {
				stale = append(stale, StaleChange{
					ExternalID: c.ExternalID,
					Reason:     "already exists (was it created since the plan?)",
				})
			}
		case OpUpdate:
			existing, exists := liveByExtID[c.ExternalID]
			if !exists {
				stale = append(stale, StaleChange{
					ExternalID: c.ExternalID,
					Reason:     "no longer exists",
				})
			} else if c.Before != nil && existing.Name != c.Before.Name {
				stale = append(stale, StaleChange{
					ExternalID: c.ExternalID,
					Reason:     fmt.Sprintf("name changed: plan expected %q, live has %q", c.Before.Name, existing.Name),
				})
			}
		case OpDelete:
			if _, exists := liveByExtID[c.ExternalID]; !exists {
				stale = append(stale, StaleChange{
					ExternalID: c.ExternalID,
					Reason:     "already deleted",
				})
			}
		}
	}
	return stale
}
