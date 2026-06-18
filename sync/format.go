package sync

import (
	"fmt"
	"io"
	"sort"
)

const (
	green  = "\033[32m"
	yellow = "\033[33m"
	red    = "\033[31m"
	cyan   = "\033[36m"
	reset  = "\033[0m"
	bold   = "\033[1m"
)

func FormatPlan(w io.Writer, plan *Plan) {
	_, _ = fmt.Fprintf(w, "Catalog: %s (%s)\n\n", plan.Catalog, plan.CatalogID)

	for _, c := range plan.Changes {
		name := EntityName(c)
		switch c.Op {
		case OpCreate:
			_, _ = fmt.Fprintf(w, "  %s+%s create  %s  %q\n", green, reset, c.ExternalID, name)
		case OpUpdate:
			_, _ = fmt.Fprintf(w, "  %s~%s update  %s  %q\n", yellow, reset, c.ExternalID, name)
			printFieldDiffs(w, c.FieldDiffs)
		case OpDelete:
			_, _ = fmt.Fprintf(w, "  %s-%s delete  %s  %q\n", red, reset, c.ExternalID, name)
		case OpNoop:
			_, _ = fmt.Fprintf(w, "  %s=%s noop    %s  %q\n", cyan, reset, c.ExternalID, name)
		}
	}

	_, _ = fmt.Fprintf(w, "\nPlan: %d to create, %d to update, %d to delete, %d unchanged.\n",
		plan.Counts.Create, plan.Counts.Update, plan.Counts.Delete, plan.Counts.Noop)
}

func printFieldDiffs(w io.Writer, diffs map[string][2]string) {
	keys := make([]string, 0, len(diffs))
	for k := range diffs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		d := diffs[k]
		_, _ = fmt.Fprintf(w, "      %s: %q → %q\n", k, d[0], d[1])
	}
}

func EntityName(c Change) string {
	if c.After != nil {
		return c.After.Name
	}
	if c.Before != nil {
		return c.Before.Name
	}
	return ""
}
