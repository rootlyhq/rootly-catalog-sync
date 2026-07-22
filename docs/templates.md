# Template syntax

Field mappings in `rootly-catalog-sync` use [Go templates](https://pkg.go.dev/text/template). Each template is evaluated against the source entry (a `map[string]any`) and must produce a string.

## Basics

Reference top-level fields with dot notation. In v2 config, all fields live directly in `map:`:

```yaml
map:
  external_id: "{{ .id }}"
  name: "{{ .name }}"
  owner: "{{ .owner }}"
  tier: "{{ .tier }}"
```

Given the source entry:
```yaml
id: payments-api
name: Payments API
owner: team-payments
tier: "1"
```

This produces:
| Field | Value |
|-------|-------|
| `external_id` | `payments-api` |
| `name` | `Payments API` |
| `owner` | `team-payments` |
| `tier` | `1` |

## Built-in functions

### `get` -- nested map access

Access nested fields safely:

```yaml
map:
  external_id: "{{ get .metadata \"service_id\" }}"
```

Source entry:
```yaml
metadata:
  service_id: svc-123
  team: platform
```

Result: `svc-123`

If the key does not exist, `get` returns an error:
```
executing template: key "missing_key" not found in map
```

### `default` -- fallback value

Return a fallback when a field is nil or empty:

```yaml
map:
  tier: "{{ default .tier \"unknown\" }}"
  region: "{{ default .region \"us-east-1\" }}"
```

| `.tier` value | Result |
|---------------|--------|
| `"1"` | `1` |
| `""` | `unknown` |
| `nil` (field missing with `default`) | `unknown` |

**Note:** `default` prevents the `missingkey=error` behavior for the wrapped field. If you want a hard error on missing fields, use `{{ .field }}` directly.

## Missing keys

Templates are compiled with `missingkey=error`. Referencing a field that does not exist in the source entry causes a hard error:

```
entry 0: evaluating external_id: executing template:
  map has no entry for key "id"
```

This is intentional -- it catches typos and schema mismatches early rather than silently producing empty values.

**To make a field optional**, wrap it with `default`:

```yaml
map:
  description: "{{ default .description \"\" }}"
```

## Static values

Templates can produce static strings:

```yaml
map:
  source: "catalog-sync"
  environment: "production"
```

No `{{ }}` delimiters needed -- the value is used as-is.

## Combining fields

Use Go template syntax to concatenate or transform:

```yaml
map:
  # Concatenation
  external_id: "{{ .org }}/{{ .name }}"

  # Conditional
  tier: "{{ if .critical }}1{{ else }}3{{ end }}"
```

## Template caching

Parsed templates are cached in memory using a `sync.Map` keyed by the template string. This means:

- The first evaluation of a template parses it; subsequent evaluations reuse the parsed version.
- Identical template strings across different outputs share the same parsed template.
- There is no cache eviction -- templates are cached for the lifetime of the process.

This is transparent and requires no configuration.

## Common patterns

### Backstage entity mapping

```yaml
sync:
  - from:
      backstage:
        url: https://backstage.internal
        token: "$(BACKSTAGE_TOKEN)"
        kind: Component
    to: Services
    map:
      external_id: "{{ get .metadata \"name\" }}"
      name: "{{ get .metadata \"name\" }}"
      kind: "{{ .kind }}"
      owner: "{{ get .spec \"owner\" }}"
      lifecycle: "{{ get .spec \"lifecycle\" }}"
      system: "{{ default (get .spec \"system\") \"\" }}"
```

### GitHub repo metadata

```yaml
sync:
  - from:
      github:
        token: "$(GITHUB_TOKEN)"
        owner: acme
        files: ["catalog.yaml"]
    to: Repositories
    map:
      external_id: "{{ .full_name }}"
      name: "{{ .name }}"
      language: "{{ default .language \"unknown\" }}"
      visibility: "{{ .visibility }}"
      default_branch: "{{ .default_branch }}"
```

### CSV with header mapping

Given a CSV:
```csv
id,name,owner,tier
payments-api,Payments API,team-payments,1
```

```yaml
sync:
  - from:
      csv:
        files: ["services.csv"]
    to: Services
    map:
      external_id: "{{ .id }}"
      name: "{{ .name }}"
      owner: "{{ .owner }}"
      tier: "{{ .tier }}"
```

CSV headers become the field names in the source entry.

### Exec source (BigQuery)

```yaml
sync:
  - from:
      exec:
        command: bq
        args: ["query", "--format=json", "SELECT service_id, display_name, team_email, sla_tier FROM services"]
    to: Services
    map:
      external_id: "{{ .service_id }}"
      name: "{{ .display_name }}"
      owner: "{{ .team_email }}"
      tier: "{{ default .sla_tier \"3\" }}"
```
