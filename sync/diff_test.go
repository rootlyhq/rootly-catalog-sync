package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rootlyhq/rootly-catalog-sync/catalog"
)

func TestDiff_CreateOnly(t *testing.T) {
	desired := []catalog.DesiredEntity{
		{ExternalID: "ext-1", Name: "Svc A", Fields: map[string]string{"tier": "1"}},
		{ExternalID: "ext-2", Name: "Svc B", Fields: map[string]string{"tier": "2"}},
	}

	plan := Diff("Services", "cat-1", nil, desired, false)

	if plan.Counts.Create != 2 {
		t.Errorf("expected 2 creates, got %d", plan.Counts.Create)
	}
	if plan.Counts.Update != 0 || plan.Counts.Delete != 0 || plan.Counts.Noop != 0 {
		t.Errorf("expected no updates/deletes/noops, got %+v", plan.Counts)
	}
	for _, c := range plan.Changes {
		if c.Op != "create" {
			t.Errorf("expected create op, got %s", c.Op)
		}
		if c.Before != nil {
			t.Error("create should have nil Before")
		}
		if c.After == nil {
			t.Error("create should have non-nil After")
		}
	}
}

func TestDiff_UpdateOnly(t *testing.T) {
	live := []catalog.LiveEntity{
		{ID: "id-1", ExternalID: "ext-1", Name: "Auth", Fields: map[string]string{"owner": "team-a"}},
	}
	desired := []catalog.DesiredEntity{
		{ExternalID: "ext-1", Name: "Auth Service", Fields: map[string]string{"owner": "team-b"}},
	}

	plan := Diff("Services", "cat-1", live, desired, false)

	if plan.Counts.Update != 1 {
		t.Fatalf("expected 1 update, got %d", plan.Counts.Update)
	}

	c := plan.Changes[0]
	if c.Op != "update" {
		t.Errorf("expected update, got %s", c.Op)
	}
	if d, ok := c.FieldDiffs["name"]; !ok {
		t.Error("expected name diff")
	} else if d[0] != "Auth" || d[1] != "Auth Service" {
		t.Errorf("unexpected name diff: %v", d)
	}
	if d, ok := c.FieldDiffs["owner"]; !ok {
		t.Error("expected owner diff")
	} else if d[0] != "team-a" || d[1] != "team-b" {
		t.Errorf("unexpected owner diff: %v", d)
	}
}

func TestDiff_DeleteOnly(t *testing.T) {
	live := []catalog.LiveEntity{
		{ID: "id-1", ExternalID: "ext-1", Name: "Old Service", Fields: map[string]string{}},
	}

	plan := Diff("Services", "cat-1", live, nil, true)

	if plan.Counts.Delete != 1 {
		t.Fatalf("expected 1 delete, got %d", plan.Counts.Delete)
	}
	c := plan.Changes[0]
	if c.Op != "delete" {
		t.Errorf("expected delete, got %s", c.Op)
	}
	if c.After != nil {
		t.Error("delete should have nil After")
	}
}

func TestDiff_NoPruneSkipsDeletes(t *testing.T) {
	live := []catalog.LiveEntity{
		{ID: "id-1", ExternalID: "ext-1", Name: "Old Service", Fields: map[string]string{}},
	}

	plan := Diff("Services", "cat-1", live, nil, false)

	if plan.Counts.Delete != 0 {
		t.Errorf("expected 0 deletes with allowPrune=false, got %d", plan.Counts.Delete)
	}
}

func TestDiff_Noop(t *testing.T) {
	live := []catalog.LiveEntity{
		{ID: "id-1", ExternalID: "ext-1", Name: "Svc", Fields: map[string]string{"tier": "1"}},
	}
	desired := []catalog.DesiredEntity{
		{ExternalID: "ext-1", Name: "Svc", Fields: map[string]string{"tier": "1"}},
	}

	plan := Diff("Services", "cat-1", live, desired, false)

	if plan.Counts.Noop != 1 {
		t.Fatalf("expected 1 noop, got %d", plan.Counts.Noop)
	}
	if plan.Changes[0].Op != "noop" {
		t.Errorf("expected noop, got %s", plan.Changes[0].Op)
	}
}

func TestDiff_Mixed(t *testing.T) {
	live := []catalog.LiveEntity{
		{ID: "id-1", ExternalID: "ext-1", Name: "Unchanged", Fields: map[string]string{"x": "1"}},
		{ID: "id-2", ExternalID: "ext-2", Name: "OldName", Fields: map[string]string{}},
		{ID: "id-3", ExternalID: "ext-3", Name: "ToDelete", Fields: map[string]string{}},
	}
	desired := []catalog.DesiredEntity{
		{ExternalID: "ext-1", Name: "Unchanged", Fields: map[string]string{"x": "1"}},
		{ExternalID: "ext-2", Name: "NewName", Fields: map[string]string{}},
		{ExternalID: "ext-4", Name: "Brand New", Fields: map[string]string{}},
	}

	plan := Diff("Services", "cat-1", live, desired, true)

	if plan.Counts.Create != 1 || plan.Counts.Update != 1 || plan.Counts.Delete != 1 || plan.Counts.Noop != 1 {
		t.Fatalf("unexpected counts: %+v", plan.Counts)
	}

	// Verify ordering: creates, updates, noops, deletes
	ops := make([]string, len(plan.Changes))
	for i, c := range plan.Changes {
		ops[i] = c.Op
	}
	expected := []string{"create", "update", "noop", "delete"}
	for i, op := range expected {
		if ops[i] != op {
			t.Errorf("position %d: expected %s, got %s (full order: %v)", i, op, ops[i], ops)
			break
		}
	}
}

func TestDiff_OnlyExternalIDEntitiesPruned(t *testing.T) {
	live := []catalog.LiveEntity{
		{ID: "id-1", ExternalID: "", Name: "Manual Entry", Fields: map[string]string{}},
		{ID: "id-2", ExternalID: "ext-2", Name: "Managed", Fields: map[string]string{}},
	}

	plan := Diff("Services", "cat-1", live, nil, true)

	if plan.Counts.Delete != 1 {
		t.Fatalf("expected 1 delete (only ext-2), got %d", plan.Counts.Delete)
	}
	if plan.Changes[0].ExternalID != "ext-2" {
		t.Errorf("expected ext-2 to be deleted, got %s", plan.Changes[0].ExternalID)
	}
}

func TestCheckSafety_EmptySource(t *testing.T) {
	plan := &Plan{Counts: Counts{Delete: 5}}
	err := CheckSafety(plan, 5, 0, DefaultPruneThreshold)
	if err == nil {
		t.Fatal("expected error for empty source")
	}
}

func TestCheckSafety_PruneRatioExceeded(t *testing.T) {
	plan := &Plan{Counts: Counts{Delete: 3}}
	err := CheckSafety(plan, 10, 7, 0.2)
	if err == nil {
		t.Fatal("expected error for prune ratio exceeded (30% > 20%)")
	}
}

func TestCheckSafety_OK(t *testing.T) {
	plan := &Plan{Counts: Counts{Delete: 1}}
	err := CheckSafety(plan, 10, 9, 0.2)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestCheckSafety_EmptyDesiredAllLive(t *testing.T) {
	// When the source returns 0 entities but live entities exist,
	// CheckSafety must refuse the operation to prevent accidental mass deletion.
	plan := &Plan{Counts: Counts{Delete: 10}}
	err := CheckSafety(plan, 10, 0, DefaultPruneThreshold)
	if err == nil {
		t.Fatal("expected error when desired count is 0 and live entities exist")
	}
	if !strings.Contains(err.Error(), "empty source") {
		t.Errorf("expected error to mention 'empty source', got: %v", err)
	}
}

func TestCheckSafety_HighPruneRatio(t *testing.T) {
	// 10 live entities, 5 to delete = 50% prune ratio, threshold is 20%.
	plan := &Plan{Counts: Counts{Delete: 5}}
	err := CheckSafety(plan, 10, 5, 0.2)
	if err == nil {
		t.Fatal("expected error when prune ratio (50%) exceeds threshold (20%)")
	}
	if !strings.Contains(err.Error(), "prune ratio 50%") {
		t.Errorf("expected error to contain 'prune ratio 50%%', got: %v", err)
	}
}

func TestCheckSafety_WithinThreshold(t *testing.T) {
	// 10 live entities, 1 to delete = 10% prune ratio, threshold is 20%.
	plan := &Plan{Counts: Counts{Delete: 1}}
	err := CheckSafety(plan, 10, 9, 0.2)
	if err != nil {
		t.Fatalf("expected no error for 10%% prune ratio within 20%% threshold, got: %v", err)
	}
}

func TestCounts_IsNoop(t *testing.T) {
	tests := []struct {
		name     string
		counts   Counts
		expected bool
	}{
		{"all zero", Counts{}, true},
		{"noop only", Counts{Noop: 5}, true},
		{"has creates", Counts{Create: 1}, false},
		{"has updates", Counts{Update: 1}, false},
		{"has deletes", Counts{Delete: 1}, false},
		{"has creates and noops", Counts{Create: 1, Noop: 3}, false},
		{"all non-zero", Counts{Create: 1, Update: 2, Delete: 3, Noop: 4}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.counts.IsNoop()
			if got != tt.expected {
				t.Errorf("Counts%+v.IsNoop() = %v, want %v", tt.counts, got, tt.expected)
			}
		})
	}
}

func TestSavePlanLoadPlan(t *testing.T) {
	original := &Plan{
		Catalog:   "Services",
		CatalogID: "cat-1",
		Changes: []Change{
			{
				Op:         "create",
				ExternalID: "ext-1",
				After:      &catalog.DesiredEntity{ExternalID: "ext-1", Name: "Svc", Fields: map[string]string{"tier": "1"}},
			},
			{
				Op:         "update",
				ExternalID: "ext-2",
				Before:     &catalog.LiveEntity{ID: "id-2", ExternalID: "ext-2", Name: "Old"},
				After:      &catalog.DesiredEntity{ExternalID: "ext-2", Name: "New"},
				FieldDiffs: map[string][2]string{"name": {"Old", "New"}},
			},
		},
		Counts: Counts{Create: 1, Update: 1},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "plan.json")

	if err := SavePlan(original, path); err != nil {
		t.Fatalf("SavePlan: %v", err)
	}

	loaded, err := LoadPlan(path)
	if err != nil {
		t.Fatalf("LoadPlan: %v", err)
	}

	if loaded.Catalog != original.Catalog || loaded.CatalogID != original.CatalogID {
		t.Errorf("catalog mismatch: %s/%s vs %s/%s", loaded.Catalog, loaded.CatalogID, original.Catalog, original.CatalogID)
	}
	if loaded.Counts != original.Counts {
		t.Errorf("counts mismatch: %+v vs %+v", loaded.Counts, original.Counts)
	}
	if len(loaded.Changes) != len(original.Changes) {
		t.Fatalf("changes length mismatch: %d vs %d", len(loaded.Changes), len(original.Changes))
	}
	for i, c := range loaded.Changes {
		if c.Op != original.Changes[i].Op || c.ExternalID != original.Changes[i].ExternalID {
			t.Errorf("change %d mismatch: %s/%s vs %s/%s", i, c.Op, c.ExternalID, original.Changes[i].Op, original.Changes[i].ExternalID)
		}
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("plan file should exist: %v", err)
	}
}

func TestDiff_ManagedFieldsOnly(t *testing.T) {
	// Live entity has fields the desired doesn't mention.
	// Those unmanaged fields should NOT cause an update.
	live := []catalog.LiveEntity{
		{ExternalID: "svc-1", Name: "Service A", Fields: map[string]string{
			"description": "desc", "color": "#fff", "pagerduty_id": "PD123",
		}},
	}
	desired := []catalog.DesiredEntity{
		{ExternalID: "svc-1", Name: "Service A", Fields: map[string]string{
			"description": "desc",
		}},
	}
	plan := Diff("test", "t1", live, desired, false)
	if plan.Counts.Noop != 1 {
		t.Errorf("expected 1 noop (unmanaged fields ignored), got %d noops, %d updates", plan.Counts.Noop, plan.Counts.Update)
	}
}

func TestDiff_ManagedFieldChanged(t *testing.T) {
	live := []catalog.LiveEntity{
		{ExternalID: "svc-1", Name: "Service A", Fields: map[string]string{
			"description": "old desc", "color": "#fff",
		}},
	}
	desired := []catalog.DesiredEntity{
		{ExternalID: "svc-1", Name: "Service A", Fields: map[string]string{
			"description": "new desc",
		}},
	}
	plan := Diff("test", "t1", live, desired, false)
	if plan.Counts.Update != 1 {
		t.Errorf("expected 1 update for managed field change, got %d", plan.Counts.Update)
	}
	if plan.Changes[0].FieldDiffs["description"] != [2]string{"old desc", "new desc"} {
		t.Errorf("unexpected diff: %v", plan.Changes[0].FieldDiffs)
	}
	// color should NOT be in diffs (not managed)
	if _, has := plan.Changes[0].FieldDiffs["color"]; has {
		t.Error("color should not appear in diffs — not a managed field")
	}
}

func TestCheckNativeSafety_SentinelEnvironment(t *testing.T) {
	plan := &Plan{Counts: Counts{Delete: 3}}
	err := CheckNativeSafety(plan, "environment", 3, 0, 0.5)
	if err == nil {
		t.Fatal("expected error refusing to delete all environments")
	}
	if !strings.Contains(err.Error(), "at least one must remain") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckNativeSafety_SentinelTeam(t *testing.T) {
	plan := &Plan{Counts: Counts{Delete: 1}}
	err := CheckNativeSafety(plan, "team", 1, 0, 0.5)
	if err == nil {
		t.Fatal("expected error refusing to delete all teams")
	}
}

func TestCheckNativeSafety_ServiceNoSentinel(t *testing.T) {
	plan := &Plan{Counts: Counts{Delete: 5}}
	// Services have no sentinel — can delete all (if prune ratio allows)
	err := CheckNativeSafety(plan, "service", 5, 0, 1.0)
	// Should fail on empty source, not sentinel
	if err == nil {
		t.Fatal("expected error for empty source")
	}
}
