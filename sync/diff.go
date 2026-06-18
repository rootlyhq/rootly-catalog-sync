package sync

import (
	"sort"

	"github.com/rootlyhq/rootly-catalog-sync/catalog"
)

func Diff(catalogName string, catalogID string, live []catalog.LiveEntity, desired []catalog.DesiredEntity, allowPrune bool) *Plan {
	liveByExtID := make(map[string]catalog.LiveEntity)
	for _, e := range live {
		if e.ExternalID != "" {
			liveByExtID[e.ExternalID] = e
		}
	}

	var creates, updates, noops, deletes []Change

	desiredExtIDs := make(map[string]bool, len(desired))
	for _, d := range desired {
		desiredExtIDs[d.ExternalID] = true
		l, exists := liveByExtID[d.ExternalID]
		if !exists {
			creates = append(creates, Change{
				Op:         OpCreate,
				ExternalID: d.ExternalID,
				After:      ptr(d),
			})
			continue
		}

		diffs := computeFieldDiffs(l, d)
		if len(diffs) > 0 {
			updates = append(updates, Change{
				Op:         OpUpdate,
				ExternalID: d.ExternalID,
				Before:     ptr2(l),
				After:      ptr(d),
				FieldDiffs: diffs,
			})
		} else {
			noops = append(noops, Change{
				Op:         OpNoop,
				ExternalID: d.ExternalID,
				Before:     ptr2(l),
				After:      ptr(d),
			})
		}
	}

	if allowPrune {
		for extID, l := range liveByExtID {
			if !desiredExtIDs[extID] {
				deletes = append(deletes, Change{
					Op:         OpDelete,
					ExternalID: extID,
					Before:     ptr2(l),
				})
			}
		}
		sort.Slice(deletes, func(i, j int) bool {
			return deletes[i].ExternalID < deletes[j].ExternalID
		})
	}

	changes := make([]Change, 0, len(creates)+len(updates)+len(noops)+len(deletes))
	changes = append(changes, creates...)
	changes = append(changes, updates...)
	changes = append(changes, noops...)
	changes = append(changes, deletes...)

	return &Plan{
		Catalog:   catalogName,
		CatalogID: catalogID,
		Changes:   changes,
		Counts: Counts{
			Create: len(creates),
			Update: len(updates),
			Delete: len(deletes),
			Noop:   len(noops),
		},
	}
}

func computeFieldDiffs(live catalog.LiveEntity, desired catalog.DesiredEntity) map[string][2]string {
	diffs := make(map[string][2]string)

	if live.Name != desired.Name {
		diffs["name"] = [2]string{live.Name, desired.Name}
	}

	allKeys := make(map[string]bool)
	for k := range live.Fields {
		allKeys[k] = true
	}
	for k := range desired.Fields {
		allKeys[k] = true
	}

	for k := range allKeys {
		lv := live.Fields[k]
		dv := desired.Fields[k]
		if lv != dv {
			diffs[k] = [2]string{lv, dv}
		}
	}

	return diffs
}

func ptr(d catalog.DesiredEntity) *catalog.DesiredEntity { return &d }
func ptr2(l catalog.LiveEntity) *catalog.LiveEntity      { return &l }
