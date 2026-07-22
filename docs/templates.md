# Template syntax

Field mappings in `rootly-catalog-sync` use [Go templates](https://pkg.go.dev/text/template). Each template is evaluated against the source entry (a `map[string]any`) and must produce a string.

## Basics

Reference top-level fields with dot notation:

```yaml
external_id: "{{ .id }}"
name: "{{ .name }}"
fields:
  owner:
    value: "{{ .owner }}"
  tier:
    value: "{{ .tier }}"
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
fields:
  tier:
    value: "{{ default .tier \"unknown\" }}"
  region:
    value: "{{ default .region \"us-east-1\" }}"
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
fields:
  description:
    value: "{{ default .description \"\" }}"
```

## Static values

Templates can produce static strings:

```yaml
fields:
  source:
    value: "catalog-sync"
  environment:
    value: "production"
```

No `{{ }}` delimiters needed -- the value is used as-is.

## Combining fields

Use Go template syntax to concatenate or transform:

```yaml
# Concatenation
external_id: "{{ .org }}/{{ .name }}"

# Conditional
fields:
  tier:
    value: "{{ if .critical }}1{{ else }}3{{ end }}"
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
outputs:
  - catalog: "Services"
    external_id: "{{ get .metadata \"name\" }}"
    name: "{{ get .metadata \"name\" }}"
    fields:
      kind:
        value: "{{ .kind }}"
      owner:
        value: "{{ get .spec \"owner\" }}"
      lifecycle:
        value: "{{ get .spec \"lifecycle\" }}"
      system:
        value: "{{ default (get .spec \"system\") \"\" }}"
```

### GitHub repo metadata

```yaml
outputs:
  - catalog: "Repositories"
    external_id: "{{ .full_name }}"
    name: "{{ .name }}"
    fields:
      language:
        value: "{{ default .language \"unknown\" }}"
      visibility:
        value: "{{ .visibility }}"
      default_branch:
        value: "{{ .default_branch }}"
```

### CSV with header mapping

Given a CSV:
```csv
id,name,owner,tier
payments-api,Payments API,team-payments,1
```

```yaml
outputs:
  - catalog: "Services"
    external_id: "{{ .id }}"
    name: "{{ .name }}"
    fields:
      owner:
        value: "{{ .owner }}"
      tier:
        value: "{{ .tier }}"
```

CSV headers become the field names in the source entry.

### Exec source (BigQuery)

```yaml
outputs:
  - catalog: "Services"
    external_id: "{{ .service_id }}"
    name: "{{ .display_name }}"
    fields:
      owner:
        value: "{{ .team_email }}"
      tier:
        value: "{{ default .sla_tier \"3\" }}"
```
