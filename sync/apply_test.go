package sync

import (
	"context"
	"errors"
	"testing"

	"github.com/rootlyhq/rootly-catalog-sync/catalog"
)

func TestApply_CreatesAndDeletes(t *testing.T) {
	var upsertedEnts []catalog.DesiredEntity
	var upsertedCatalog string
	var deletedIDs []string
	var deletedCatalog string

	applier := &Applier{
		BulkUpsert: func(ctx context.Context, catalogID string, ents []catalog.DesiredEntity) error {
			upsertedCatalog = catalogID
			upsertedEnts = ents
			return nil
		},
		BulkDelete: func(ctx context.Context, catalogID string, externalIDs []string) error {
			deletedCatalog = catalogID
			deletedIDs = externalIDs
			return nil
		},
	}

	plan := &Plan{
		CatalogID: "cat-123",
		Changes: []Change{
			{
				Op:         OpCreate,
				ExternalID: "svc-a",
				After:      &catalog.DesiredEntity{ExternalID: "svc-a", Name: "Service A"},
			},
			{
				Op:         OpCreate,
				ExternalID: "svc-b",
				After:      &catalog.DesiredEntity{ExternalID: "svc-b", Name: "Service B"},
			},
			{
				Op:         OpDelete,
				ExternalID: "svc-old",
			},
		},
	}

	if err := applier.Apply(context.Background(), plan); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if upsertedCatalog != "cat-123" {
		t.Errorf("BulkUpsert catalog = %q, want %q", upsertedCatalog, "cat-123")
	}
	if len(upsertedEnts) != 2 {
		t.Fatalf("BulkUpsert got %d entities, want 2", len(upsertedEnts))
	}
	if upsertedEnts[0].ExternalID != "svc-a" {
		t.Errorf("upsertedEnts[0].ExternalID = %q, want %q", upsertedEnts[0].ExternalID, "svc-a")
	}
	if upsertedEnts[1].ExternalID != "svc-b" {
		t.Errorf("upsertedEnts[1].ExternalID = %q, want %q", upsertedEnts[1].ExternalID, "svc-b")
	}

	if deletedCatalog != "cat-123" {
		t.Errorf("BulkDelete catalog = %q, want %q", deletedCatalog, "cat-123")
	}
	if len(deletedIDs) != 1 || deletedIDs[0] != "svc-old" {
		t.Errorf("BulkDelete got IDs %v, want [svc-old]", deletedIDs)
	}
}

func TestApply_UpdateIncludedInUpserts(t *testing.T) {
	var upsertedEnts []catalog.DesiredEntity

	applier := &Applier{
		BulkUpsert: func(ctx context.Context, catalogID string, ents []catalog.DesiredEntity) error {
			upsertedEnts = ents
			return nil
		},
		BulkDelete: func(ctx context.Context, catalogID string, externalIDs []string) error {
			t.Error("BulkDelete should not be called for update-only plan")
			return nil
		},
	}

	plan := &Plan{
		CatalogID: "cat-456",
		Changes: []Change{
			{
				Op:         OpUpdate,
				ExternalID: "svc-x",
				After:      &catalog.DesiredEntity{ExternalID: "svc-x", Name: "Updated X"},
			},
		},
	}

	if err := applier.Apply(context.Background(), plan); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(upsertedEnts) != 1 {
		t.Fatalf("BulkUpsert got %d entities, want 1", len(upsertedEnts))
	}
	if upsertedEnts[0].Name != "Updated X" {
		t.Errorf("upsertedEnts[0].Name = %q, want %q", upsertedEnts[0].Name, "Updated X")
	}
}

func TestApply_NoopPlan(t *testing.T) {
	applier := &Applier{
		BulkUpsert: func(ctx context.Context, catalogID string, ents []catalog.DesiredEntity) error {
			t.Error("BulkUpsert should not be called for noop plan")
			return nil
		},
		BulkDelete: func(ctx context.Context, catalogID string, externalIDs []string) error {
			t.Error("BulkDelete should not be called for noop plan")
			return nil
		},
	}

	plan := &Plan{
		CatalogID: "cat-789",
		Changes: []Change{
			{Op: OpNoop, ExternalID: "svc-1"},
			{Op: OpNoop, ExternalID: "svc-2"},
		},
	}

	if err := applier.Apply(context.Background(), plan); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApply_UpsertError(t *testing.T) {
	upsertErr := errors.New("api unavailable")

	applier := &Applier{
		BulkUpsert: func(ctx context.Context, catalogID string, ents []catalog.DesiredEntity) error {
			return upsertErr
		},
		BulkDelete: func(ctx context.Context, catalogID string, externalIDs []string) error {
			t.Error("BulkDelete should not be called when BulkUpsert fails")
			return nil
		},
	}

	plan := &Plan{
		CatalogID: "cat-err",
		Changes: []Change{
			{
				Op:         OpCreate,
				ExternalID: "svc-fail",
				After:      &catalog.DesiredEntity{ExternalID: "svc-fail", Name: "Fail"},
			},
			{
				Op:         OpDelete,
				ExternalID: "svc-del",
			},
		},
	}

	err := applier.Apply(context.Background(), plan)
	if err == nil {
		t.Fatal("expected error from Apply, got nil")
	}
	if !errors.Is(err, upsertErr) {
		t.Errorf("expected wrapped upsertErr, got: %v", err)
	}
}

func TestApply_DeleteError(t *testing.T) {
	deleteErr := errors.New("delete failed")

	applier := &Applier{
		BulkUpsert: func(ctx context.Context, catalogID string, ents []catalog.DesiredEntity) error {
			return nil
		},
		BulkDelete: func(ctx context.Context, catalogID string, externalIDs []string) error {
			return deleteErr
		},
	}

	plan := &Plan{
		CatalogID: "cat-err2",
		Changes: []Change{
			{
				Op:         OpDelete,
				ExternalID: "svc-del",
			},
		},
	}

	err := applier.Apply(context.Background(), plan)
	if err == nil {
		t.Fatal("expected error from Apply, got nil")
	}
	if !errors.Is(err, deleteErr) {
		t.Errorf("expected wrapped deleteErr, got: %v", err)
	}
}
