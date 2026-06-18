package sync

import (
	"testing"

	"github.com/rootlyhq/rootly-catalog-sync/catalog"
)

func TestValidatePlanFreshness_Clean(t *testing.T) {
	plan := &Plan{
		Changes: []Change{
			{Op: OpCreate, ExternalID: "new-svc"},
			{Op: OpUpdate, ExternalID: "existing-svc", Before: &catalog.LiveEntity{Name: "Old Name"}},
			{Op: OpDelete, ExternalID: "gone-svc"},
		},
	}
	live := []catalog.LiveEntity{
		{ExternalID: "existing-svc", Name: "Old Name"},
		{ExternalID: "gone-svc", Name: "Gone"},
	}

	stale := ValidatePlanFreshness(plan, live)
	if len(stale) != 0 {
		t.Fatalf("expected 0 stale changes, got %d: %+v", len(stale), stale)
	}
}

func TestValidatePlanFreshness_CreateAlreadyExists(t *testing.T) {
	plan := &Plan{
		Changes: []Change{
			{Op: OpCreate, ExternalID: "new-svc"},
		},
	}
	live := []catalog.LiveEntity{
		{ExternalID: "new-svc", Name: "Already Here"},
	}

	stale := ValidatePlanFreshness(plan, live)
	if len(stale) != 1 {
		t.Fatalf("expected 1 stale change, got %d", len(stale))
	}
	if stale[0].ExternalID != "new-svc" {
		t.Errorf("expected external_id %q, got %q", "new-svc", stale[0].ExternalID)
	}
}

func TestValidatePlanFreshness_UpdateGone(t *testing.T) {
	plan := &Plan{
		Changes: []Change{
			{Op: OpUpdate, ExternalID: "svc-1", Before: &catalog.LiveEntity{Name: "Svc One"}},
		},
	}
	var live []catalog.LiveEntity // entity no longer exists

	stale := ValidatePlanFreshness(plan, live)
	if len(stale) != 1 {
		t.Fatalf("expected 1 stale change, got %d", len(stale))
	}
	if stale[0].Reason != "no longer exists" {
		t.Errorf("unexpected reason: %s", stale[0].Reason)
	}
}

func TestValidatePlanFreshness_DeleteAlreadyGone(t *testing.T) {
	plan := &Plan{
		Changes: []Change{
			{Op: OpDelete, ExternalID: "old-svc"},
		},
	}
	var live []catalog.LiveEntity // entity already deleted

	stale := ValidatePlanFreshness(plan, live)
	if len(stale) != 1 {
		t.Fatalf("expected 1 stale change, got %d", len(stale))
	}
	if stale[0].Reason != "already deleted" {
		t.Errorf("unexpected reason: %s", stale[0].Reason)
	}
}

func TestValidatePlanFreshness_UpdateNameChanged(t *testing.T) {
	plan := &Plan{
		Changes: []Change{
			{Op: OpUpdate, ExternalID: "svc-1", Before: &catalog.LiveEntity{Name: "Original"}},
		},
	}
	live := []catalog.LiveEntity{
		{ExternalID: "svc-1", Name: "Renamed"},
	}

	stale := ValidatePlanFreshness(plan, live)
	if len(stale) != 1 {
		t.Fatalf("expected 1 stale change, got %d", len(stale))
	}
	if stale[0].ExternalID != "svc-1" {
		t.Errorf("expected external_id %q, got %q", "svc-1", stale[0].ExternalID)
	}
	want := `name changed: plan expected "Original", live has "Renamed"`
	if stale[0].Reason != want {
		t.Errorf("unexpected reason:\n  got:  %s\n  want: %s", stale[0].Reason, want)
	}
}
