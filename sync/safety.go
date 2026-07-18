package sync

import "fmt"

const DefaultPruneThreshold = 0.2

// CheckSafety validates a plan against safety guards.
// minEntities > 0 prevents deleting all records (e.g. environments/teams must keep ≥1).
func CheckSafety(plan *Plan, liveCount int, desiredCount int, pruneThreshold float64, minEntities ...int) error {
	minKeep := 0
	if len(minEntities) > 0 {
		minKeep = minEntities[0]
	}

	if minKeep > 0 && liveCount > 0 {
		if desiredCount == 0 {
			return fmt.Errorf("refusing to delete all entries — at least %d must remain", minKeep)
		}
		if plan.Counts.Delete >= liveCount {
			return fmt.Errorf("refusing to delete all entries — at least %d must remain", minKeep)
		}
	}

	if desiredCount == 0 && liveCount > 0 {
		return fmt.Errorf("empty source: refusing to delete all %d live entities — source returned 0 entries", liveCount)
	}

	if liveCount > 0 && plan.Counts.Delete > 0 {
		ratio := float64(plan.Counts.Delete) / float64(liveCount)
		if ratio > pruneThreshold {
			return fmt.Errorf(
				"prune ratio %.0f%% (%d/%d) exceeds threshold %.0f%% — reduce deletions or raise the threshold",
				ratio*100, plan.Counts.Delete, liveCount, pruneThreshold*100,
			)
		}
	}

	return nil
}
