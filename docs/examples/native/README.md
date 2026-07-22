# Native Resources Example

Sync services, functionalities, environments, and teams directly to Rootly's native resources.

## Config

A single config can sync multiple resource types using separate pipelines:

```yaml
version: 1
sync_id: native-resources
pipelines:
  - sources:
      - local:
          files: ["services.yaml"]
    outputs:
      - type: service
        external_id: "{{ .id }}"
        name: "{{ .name }}"
        fields:
          description:
            value: "{{ .description }}"
          pagerduty_id:
            value: "{{ .pagerduty_id }}"

  - sources:
      - local:
          files: ["teams.yaml"]
    outputs:
      - type: team
        external_id: "{{ .id }}"
        name: "{{ .name }}"
        fields:
          description:
            value: "{{ .description }}"
          color:
            value: "{{ .color }}"

  - sources:
      - local:
          files: ["environments.yaml"]
    outputs:
      - type: environment
        external_id: "{{ .id }}"
        name: "{{ .name }}"
        fields:
          description:
            value: "{{ .description }}"
          color:
            value: "{{ .color }}"
```

## Usage

```bash
export ROOTLY_API_KEY=rootly_...
rootly-catalog-sync plan    # preview all resource types
rootly-catalog-sync sync    # apply
rootly-catalog-sync status  # verify
```

## Resource types

| Type | `type` value | Sentinel | Endpoint |
|------|-------------|----------|----------|
| Services | `service` | No | `/v1/services/bulk_upsert` |
| Functionalities | `functionality` | No | `/v1/functionalities/bulk_upsert` |
| Environments | `environment` | ≥1 must remain | `/v1/environments/bulk_upsert` |
| Teams | `team` | ≥1 must remain | `/v1/teams/bulk_upsert` |

## Built-in fields per type

**Services:** `description`, `color`, `backstage_id`, `cortex_id`, `opsgenie_id`, `opsgenie_team_id`, `opslevel_id`, `pagerduty_id`, `service_now_ci_sys_id`, `github_repository_name`, `github_repository_branch`, `gitlab_repository_name`, `gitlab_repository_branch`, `kubernetes_deployment_name`, `alerts_email_enabled`

**Functionalities:** `description`, `color`, `backstage_id`, `cortex_id`, `opsgenie_id`, `opsgenie_team_id`, `opslevel_id`, `pagerduty_id`, `service_now_ci_sys_id`

**Environments:** `description`, `color`, `position`

**Teams:** `description`, `color`, `backstage_id`, `cortex_id`, `opsgenie_id`, `opslevel_id`, `pagerduty_id`, `pagerduty_service_id`, `pagertree_id`, `victor_ops_id`, `service_now_ci_sys_id`, `alerts_email_enabled`

## Custom properties

Native resources also support custom catalog properties. Text properties are auto-created; other kinds (reference, group, slack_channel, etc.) must be created in the Rootly UI first.

### Text properties

Fields not in the built-in list are auto-created as text properties:

```yaml
pipelines:
  - sources:
      - local:
          files: ["services.yaml"]
    outputs:
      - type: service
        external_id: "{{ .id }}"
        name: "{{ .name }}"
        fields:
          description:
            value: "{{ .description }}"
          changelog-url:
            value: "{{ .changelog_url }}"
```

### Reference properties

Reference fields point to entries in a catalog. Define the reference data in a separate file, sync it as a catalog in an earlier pipeline, then reference it by name:

```yaml
# tiers.yml
- id: tier-1
  name: "Tier 1"
- id: tier-2
  name: "Tier 2"
- id: tier-3
  name: "Tier 3"
```

```yaml
# rootly-catalog-sync.yaml
version: 1
sync_id: native-with-refs
pipelines:
  # Pipeline 1: sync reference data
  - sources:
      - local:
          files: ["tiers.yml"]
    outputs:
      - catalog: "Tiers"
        external_id: "{{ .id }}"
        name: "{{ .name }}"

  # Pipeline 2: sync services with tier reference
  - sources:
      - local:
          files: ["services.yaml"]
    outputs:
      - type: service
        external_id: "{{ .id }}"
        name: "{{ .name }}"
        fields:
          description:
            value: "{{ .description }}"
          service-tier:
            value: "{{ .tier }}"
            kind: reference
            catalog: "Tiers"
```

The `service-tier` property must exist in the Rootly UI as a `reference` kind pointing to the "Tiers" catalog. The tool resolves the human-readable name ("Tier 1") to the catalog entity UUID automatically.

Pipeline order matters — reference catalogs must be synced before the resources that use them.
