package source

import "context"

type Entry map[string]any

type Source interface {
	Name() string
	Load(ctx context.Context) ([]Entry, error)
}
