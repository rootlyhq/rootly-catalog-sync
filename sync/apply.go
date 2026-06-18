package sync

import (
	"context"
	"fmt"

	"github.com/rootlyhq/rootly-catalog-sync/catalog"
)

type Applier struct {
	BulkUpsert func(ctx context.Context, catalogID string, ents []catalog.DesiredEntity) error
	BulkDelete func(ctx context.Context, catalogID string, externalIDs []string) error
}

func (a *Applier) Apply(ctx context.Context, plan *Plan) error {
	var upserts []catalog.DesiredEntity
	var deleteIDs []string

	for _, c := range plan.Changes {
		switch c.Op {
		case OpCreate, OpUpdate:
			if c.After != nil {
				upserts = append(upserts, *c.After)
			}
		case OpDelete:
			deleteIDs = append(deleteIDs, c.ExternalID)
		}
	}

	if len(upserts) > 0 {
		if err := a.BulkUpsert(ctx, plan.CatalogID, upserts); err != nil {
			return fmt.Errorf("bulk upsert: %w", err)
		}
	}

	if len(deleteIDs) > 0 {
		if err := a.BulkDelete(ctx, plan.CatalogID, deleteIDs); err != nil {
			return fmt.Errorf("bulk delete: %w", err)
		}
	}

	return nil
}
