package sync

import "github.com/rootlyhq/rootly-catalog-sync/catalog"

const (
	OpCreate = "create"
	OpUpdate = "update"
	OpDelete = "delete"
	OpNoop   = "noop"
)

type Change struct {
	Op         string
	ExternalID string
	Before     *catalog.LiveEntity
	After      *catalog.DesiredEntity
	FieldDiffs map[string][2]string
}

type Plan struct {
	Catalog   string
	CatalogID string
	Changes   []Change
	Counts    Counts
}

type Counts struct {
	Create int
	Update int
	Delete int
	Noop   int
}

func (c Counts) IsNoop() bool {
	return c.Create == 0 && c.Update == 0 && c.Delete == 0
}
