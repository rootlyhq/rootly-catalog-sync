package sync

import "fmt"

const DefaultPruneThreshold = 0.2

func CheckSafety(plan *Plan, liveCount int, desiredCount int, pruneThreshold float64) error {
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
