package sync

import (
	"encoding/json"
	"testing"

	"github.com/rootlyhq/rootly-catalog-sync/catalog"
)

func TestPlanToJSON(t *testing.T) {
	plan := &Plan{
		Catalog:   "Services",
		CatalogID: "cat-123",
		Counts:    Counts{Create: 1, Update: 1, Delete: 1, Noop: 0},
		Changes: []Change{
			{
				Op:         OpCreate,
				ExternalID: "ext-1",
				After:      &catalog.DesiredEntity{ExternalID: "ext-1", Name: "New Service", Fields: map[string]string{"tier": "1"}},
			},
			{
				Op:         OpUpdate,
				ExternalID: "ext-2",
				Before:     &catalog.LiveEntity{ID: "id-2", ExternalID: "ext-2", Name: "Old Name"},
				After:      &catalog.DesiredEntity{ExternalID: "ext-2", Name: "New Name"},
				FieldDiffs: map[string][2]string{"name": {"Old Name", "New Name"}},
			},
			{
				Op:         OpDelete,
				ExternalID: "ext-3",
				Before:     &catalog.LiveEntity{ID: "id-3", ExternalID: "ext-3", Name: "Gone Service"},
			},
		},
	}

	data, err := PlanToJSON(plan)
	if err != nil {
		t.Fatalf("PlanToJSON: %v", err)
	}

	var out JSONOutput
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out.Catalog != "Services" {
		t.Errorf("expected catalog=Services, got %s", out.Catalog)
	}
	if out.CatalogID != "cat-123" {
		t.Errorf("expected catalog_id=cat-123, got %s", out.CatalogID)
	}
	if out.Counts.Create != 1 || out.Counts.Update != 1 || out.Counts.Delete != 1 {
		t.Errorf("unexpected counts: %+v", out.Counts)
	}
	if len(out.Changes) != 3 {
		t.Fatalf("expected 3 changes, got %d", len(out.Changes))
	}

	// Verify create change
	c := out.Changes[0]
	if c.Op != "create" || c.ExternalID != "ext-1" || c.Name != "New Service" {
		t.Errorf("unexpected create change: %+v", c)
	}
	if c.FieldDiffs != nil {
		t.Errorf("create should have no field_diffs, got %v", c.FieldDiffs)
	}

	// Verify update change
	c = out.Changes[1]
	if c.Op != "update" || c.ExternalID != "ext-2" || c.Name != "New Name" {
		t.Errorf("unexpected update change: %+v", c)
	}
	if c.FieldDiffs == nil {
		t.Fatal("update should have field_diffs")
	}
	if d, ok := c.FieldDiffs["name"]; !ok {
		t.Error("expected name diff")
	} else if d[0] != "Old Name" || d[1] != "New Name" {
		t.Errorf("unexpected name diff: %v", d)
	}

	// Verify delete change
	c = out.Changes[2]
	if c.Op != "delete" || c.ExternalID != "ext-3" || c.Name != "Gone Service" {
		t.Errorf("unexpected delete change: %+v", c)
	}
}

func TestPlanToJSON_EmptyPlan(t *testing.T) {
	plan := &Plan{
		Catalog:   "Teams",
		CatalogID: "cat-456",
		Counts:    Counts{},
		Changes:   nil,
	}

	data, err := PlanToJSON(plan)
	if err != nil {
		t.Fatalf("PlanToJSON: %v", err)
	}

	var out JSONOutput
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out.Catalog != "Teams" {
		t.Errorf("expected catalog=Teams, got %s", out.Catalog)
	}
	if out.Changes != nil {
		t.Errorf("expected nil changes for empty plan, got %v", out.Changes)
	}
}
