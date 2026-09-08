# Backstage example: sync from Backstage catalog

This example syncs services from a [Backstage](https://backstage.io/) catalog into Rootly.

## Upgrading to v0.4.0

The generated `backstage_id` now uses Backstage's canonical lowercase reference:
`Component:Production/Payments-API` becomes `component:production/payments-api`.
The separate `kind`, `namespace`, and `name` source fields retain their original case.

If your configuration uses `external_id: "{{ .backstage_id }}"`, this changes the
identity used for matching existing Rootly entities. A sync can create replacements,
and `--allow-prune` can delete the old entities. Run `plan` and inspect the changes
before syncing after the upgrade.

To preserve the previous external IDs while using the corrected `backstage_id`
attribute, reconstruct the old reference from the unchanged source fields:

```yaml
map:
  external_id: '{{ printf "%s:%s/%s" .kind .namespace .name }}'
  name: "{{ .name }}"
  backstage_id: "{{ .backstage_id }}"
```

Configurations with external IDs independent of `backstage_id` keep their existing
identities and receive only an attribute update where `backstage_id` is mapped.

## File structure

```
docs/examples/backstage/
├── README.md                      # this file
└── rootly-catalog-sync.yaml       # config
```

## Prerequisites

- A running Backstage instance with the Catalog API enabled.
- A Backstage API token (or service-to-service auth) stored in `BACKSTAGE_TOKEN`.

## Config

`rootly-catalog-sync.yaml` connects to Backstage and maps entity fields:

```yaml
version: 2

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
      type: "{{ default (get .spec \"type\") \"\" }}"
```

<details>
<summary>v1 format (still supported)</summary>

```yaml
version: 1
sync_id: backstage-services
pipelines:
  - sources:
      - backstage:
          url: https://backstage.internal
          token: "$(BACKSTAGE_TOKEN)"
          kind: Component
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
          type:
            value: "{{ default (get .spec \"type\") \"\" }}"
```
</details>

## How it works

1. The `backstage` source fetches all entities of `kind: Component` from the Backstage Catalog API.
2. Each Backstage entity is flattened into a source entry with top-level keys: `kind`, `metadata`, `spec`, etc.
3. The `get` function accesses nested maps (`metadata.name`, `spec.owner`).
4. The `default` function provides fallbacks for optional fields like `spec.type`.

## Field mapping reference

| Backstage field | Template | Description |
|----------------|----------|-------------|
| `metadata.name` | `{{ get .metadata "name" }}` | Entity name (unique within kind+namespace) |
| `metadata.namespace` | `{{ default (get .metadata "namespace") "default" }}` | Namespace (usually "default") |
| `kind` | `{{ .kind }}` | Entity kind (Component, API, System, etc.) |
| `spec.owner` | `{{ get .spec "owner" }}` | Owning group or user |
| `spec.lifecycle` | `{{ get .spec "lifecycle" }}` | Lifecycle stage (production, experimental, deprecated) |
| `spec.type` | `{{ get .spec "type" }}` | Component type (service, website, library) |
| `spec.system` | `{{ default (get .spec "system") "" }}` | Parent system |

## Usage

```bash
export ROOTLY_API_KEY=rootly_...
export BACKSTAGE_TOKEN=...

# Preview
rootly-catalog-sync plan --dry-run --config=docs/examples/backstage/rootly-catalog-sync.yaml

# Apply
rootly-catalog-sync sync --config=docs/examples/backstage/rootly-catalog-sync.yaml
```

## Filtering by kind

The `kind` setting is a shorthand used when neither `filter` nor `filters` is set.

## Filtering multiple types in one sync

Use `filters` to select several entity types for a single Rootly target:

```yaml
version: 2
sync:
  - from:
      backstage:
        url: https://backstage.internal
        token: "$(BACKSTAGE_TOKEN)"
        filters:
          - "kind=component,spec.type=service"
          - "kind=component,spec.type=website"
          - "kind=component,spec.type=cronjob"
    to: service
    map:
      external_id: "{{ .backstage_id }}"
      name: "{{ .name }}"
      backstage_id: "{{ .backstage_id }}"
```

Each item becomes a separate `filter` query parameter on every page of the
[Backstage API request](https://backstage.io/docs/features/software-catalog/api/catalog/).
The API ORs those filter sets; comma-separated conditions within a set are ANDed.
The source loads the complete result before reconciliation, so pruning compares
against all selected types together. Keep selections for the same managed target
in one sync entry when pruning.

The existing scalar `filter` remains supported. Use either `filter` or `filters`;
configuring both is an error, as is an empty or whitespace-only item in `filters`.
An empty list behaves as if `filters` were omitted. Explicit filters override
the `kind` shorthand, so include `kind=component` in each set when needed.
The `filters` list also works in v1 sources and JSON, Jsonnet, and HCL configs.

## Syncing different kinds to separate catalogs

To sync multiple entity kinds into separate catalogs, add multiple sync entries:

```yaml
version: 2

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
      owner: "{{ get .spec \"owner\" }}"

  - from:
      backstage:
        url: https://backstage.internal
        token: "$(BACKSTAGE_TOKEN)"
        kind: API
    to: APIs
    map:
      external_id: "{{ get .metadata \"name\" }}"
      name: "{{ get .metadata \"name\" }}"
      owner: "{{ get .spec \"owner\" }}"
```

<details>
<summary>v1 format (still supported)</summary>

```yaml
pipelines:
  - sources:
      - backstage:
          url: https://backstage.internal
          token: "$(BACKSTAGE_TOKEN)"
          kind: Component
    outputs:
      - catalog: "Services"
        external_id: "{{ get .metadata \"name\" }}"
        name: "{{ get .metadata \"name\" }}"
        fields:
          owner:
            value: "{{ get .spec \"owner\" }}"

  - sources:
      - backstage:
          url: https://backstage.internal
          token: "$(BACKSTAGE_TOKEN)"
          kind: API
    outputs:
      - catalog: "APIs"
        external_id: "{{ get .metadata \"name\" }}"
        name: "{{ get .metadata \"name\" }}"
        fields:
          owner:
            value: "{{ get .spec \"owner\" }}"
```
</details>
