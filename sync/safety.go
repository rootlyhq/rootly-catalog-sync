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

// CheckNativeSafety applies extra guards for native resource types where the
// API requires at least one record to remain (environments and teams).
func CheckNativeSafety(plan *Plan, resourceType string, liveCount int, desiredCount int, pruneThreshold float64) error {
	if (resourceType == "environment" || resourceType == "team") && desiredCount == 0 && liveCount > 0 {
		return fmt.Errorf("refusing to delete all %ss — at least one must remain", resourceType)
	}
	if (resourceType == "environment" || resourceType == "team") && liveCount > 0 && plan.Counts.Delete >= liveCount {
		return fmt.Errorf("refusing to delete all %ss — at least one must remain", resourceType)
	}
	return CheckSafety(plan, liveCount, desiredCount, pruneThreshold)
}
