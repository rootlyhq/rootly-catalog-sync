# rootly-catalog-sync

Go CLI that syncs external catalog sources into Rootly's Catalog API.

## Build & Test

```bash
go build ./cmd/rootly-catalog-sync
go test ./...
golangci-lint run ./...
```

## Architecture

- `catalog/` — shared entity types (DesiredEntity, LiveEntity)
- `client/` — Rootly API client (JSON:API, bulk upsert/delete, retry)
- `config/` — YAML/Jsonnet/HCL config loading + validation
- `source/` — pluggable source connectors (inline, local, github, exec, csv, backstage, graphql)
- `tmpl/` — Go template evaluation with caching
- `mapping/` — maps source entries to catalog entities via templates
- `sync/` — diff engine, safety guards, plan files
- `tui/` — Bubble Tea interactive plan/apply UI
- `oauth/` — OAuth 2.0 (PKCE, token refresh, transport)
- `authconfig/` — user auth config (~/.rootly-catalog-sync/config.yaml)
- `cmd/rootly-catalog-sync/commands/` — CLI commands (cobra)

## Key Patterns

- `reconcileAll()` in commands/helpers.go orchestrates the full pipeline
- `catalog.DesiredEntity` / `catalog.LiveEntity` are the shared entity types
- Client uses `/v1/` API path (configurable via ROOTLY_API_URL + ROOTLY_API_PATH)
- Entity properties come back as UUIDs — client resolves via field list
- Safety: empty source bail, prune ratio threshold, delete-last ordering, sentinel (env/team ≥1)
- Native resources (service/functionality/environment/team) use `client/native.go`
- Output `type` field routes to native bulk endpoints vs catalog entity endpoints
- Template caching via sync.Map in tmpl package
- Config `fields` accepts both scalar (`slug: "{{ .x }}"`) and object (`slug: {value: "{{ .x }}", kind: reference, catalog: "Tiers"}`) forms
- Native resources support custom catalog properties via `/v1/{type}/properties` endpoints; text properties auto-created on sync, non-text require UI setup
- Reference fields resolve human-readable names to catalog entity UUIDs; referenced catalogs must exist (synced in earlier pipeline via `withApplyEach`)
- `withApplyEach` callback applies each pipeline's results before the next pipeline runs (used by sync/watch); plan/status only read

## Environment

- `ROOTLY_API_KEY` — API key (or use `login` for OAuth)
- Auth priority: ROOTLY_API_KEY env var → OAuth tokens from ~/.rootly-catalog-sync/config.yaml
- `ROOTLY_API_URL` — override base URL (default: https://api.rootly.com)
- `ROOTLY_API_PATH` — override API path prefix (default: /v1)

## CI

GitHub Actions with SHA-pinned actions. Three jobs: test, govulncheck, golangci-lint.
