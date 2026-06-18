# Backstage example: sync from Backstage catalog

This example syncs services from a [Backstage](https://backstage.io/) catalog into Rootly.

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
          kind: "{{ .kind }}"
          owner: "{{ get .spec \"owner\" }}"
          lifecycle: "{{ get .spec \"lifecycle\" }}"
          type: "{{ default (get .spec \"type\") \"\" }}"
```

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

To sync multiple entity kinds into separate catalogs, add multiple pipelines:

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
          owner: "{{ get .spec \"owner\" }}"

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
          owner: "{{ get .spec \"owner\" }}"
```
