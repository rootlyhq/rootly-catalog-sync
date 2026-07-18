package sync

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rootlyhq/rootly-catalog-sync/catalog"
)

func TestFormatPlan(t *testing.T) {
	plan := &Plan{
		Catalog:   "services",
		CatalogID: "cat-001",
		Changes: []Change{
			{
				Op:         OpCreate,
				ExternalID: "svc-new",
				After:      &catalog.DesiredEntity{ExternalID: "svc-new", Name: "New Service"},
			},
			{
				Op:         OpUpdate,
				ExternalID: "svc-upd",
				Before:     &catalog.LiveEntity{ExternalID: "svc-upd", Name: "Old Name"},
				After:      &catalog.DesiredEntity{ExternalID: "svc-upd", Name: "Updated Name"},
				FieldDiffs: map[string][2]string{
					"tier": {"gold", "platinum"},
				},
			},
			{
				Op:         OpDelete,
				ExternalID: "svc-del",
				Before:     &catalog.LiveEntity{ExternalID: "svc-del", Name: "Deleted Service"},
			},
		},
		Counts: Counts{Create: 1, Update: 1, Delete: 1, Noop: 0},
	}

	var buf bytes.Buffer
	FormatPlan(&buf, plan)
	output := buf.String()

	// Verify catalog header
	if !strings.Contains(output, "services") {
		t.Error("output missing catalog name 'services'")
	}
	if !strings.Contains(output, "cat-001") {
		t.Error("output missing catalog ID 'cat-001'")
	}

	// Verify create entry
	if !strings.Contains(output, "create") {
		t.Error("output missing 'create' label")
	}
	if !strings.Contains(output, "svc-new") {
		t.Error("output missing create external ID 'svc-new'")
	}

	// Verify update entry
	if !strings.Contains(output, "update") {
		t.Error("output missing 'update' label")
	}
	if !strings.Contains(output, "svc-upd") {
		t.Error("output missing update external ID 'svc-upd'")
	}
	// Verify field diff is printed
	if !strings.Contains(output, "tier") {
		t.Error("output missing field diff key 'tier'")
	}
	if !strings.Contains(output, "gold") {
		t.Error("output missing field diff old value 'gold'")
	}
	if !strings.Contains(output, "platinum") {
		t.Error("output missing field diff new value 'platinum'")
	}

	// Verify delete entry
	if !strings.Contains(output, "delete") {
		t.Error("output missing 'delete' label")
	}
	if !strings.Contains(output, "svc-del") {
		t.Error("output missing delete external ID 'svc-del'")
	}

	// Verify summary line
	if !strings.Contains(output, "1 to create") {
		t.Error("output missing '1 to create' in summary")
	}
	if !strings.Contains(output, "1 to update") {
		t.Error("output missing '1 to update' in summary")
	}
	if !strings.Contains(output, "1 to delete") {
		t.Error("output missing '1 to delete' in summary")
	}
}

func TestFormatPlan_NoopOnly(t *testing.T) {
	plan := &Plan{
		Catalog:   "teams",
		CatalogID: "cat-noop",
		Changes: []Change{
			{Op: OpNoop, ExternalID: "team-a", After: &catalog.DesiredEntity{Name: "Team A"}},
		},
		Counts: Counts{Noop: 1},
	}

	var buf bytes.Buffer
	FormatPlan(&buf, plan)
	output := buf.String()

	if !strings.Contains(output, "noop") {
		t.Error("output missing 'noop' label")
	}
	if !strings.Contains(output, "team-a") {
		t.Error("output missing noop external ID 'team-a'")
	}
	if !strings.Contains(output, "1 unchanged") {
		t.Error("output missing '1 unchanged' in summary")
	}
}

func TestEntityName_AfterSet(t *testing.T) {
	c := Change{
		After: &catalog.DesiredEntity{Name: "After Name"},
	}
	if got := EntityName(c); got != "After Name" {
		t.Errorf("EntityName() = %q, want %q", got, "After Name")
	}
}

func TestEntityName_BeforeOnly(t *testing.T) {
	c := Change{
		Before: &catalog.LiveEntity{Name: "Before Name"},
	}
	if got := EntityName(c); got != "Before Name" {
		t.Errorf("EntityName() = %q, want %q", got, "Before Name")
	}
}

func TestEntityName_BothSet(t *testing.T) {
	// After takes precedence
	c := Change{
		Before: &catalog.LiveEntity{Name: "Before"},
		After:  &catalog.DesiredEntity{Name: "After"},
	}
	if got := EntityName(c); got != "After" {
		t.Errorf("EntityName() = %q, want %q (After should take precedence)", got, "After")
	}
}

func TestEntityName_BothNil(t *testing.T) {
	c := Change{}
	if got := EntityName(c); got != "" {
		t.Errorf("EntityName() = %q, want empty string", got)
	}
}
