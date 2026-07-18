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
          description: "{{ .description }}"
          pagerduty_id: "{{ .pagerduty_id }}"

  - sources:
      - local:
          files: ["teams.yaml"]
    outputs:
      - type: team
        external_id: "{{ .id }}"
        name: "{{ .name }}"
        fields:
          description: "{{ .description }}"
          color: "{{ .color }}"

  - sources:
      - local:
          files: ["environments.yaml"]
    outputs:
      - type: environment
        external_id: "{{ .id }}"
        name: "{{ .name }}"
        fields:
          description: "{{ .description }}"
          color: "{{ .color }}"
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

## Supported fields per type

**Services:** `description`, `color`, `backstage_id`, `cortex_id`, `opsgenie_id`, `opsgenie_team_id`, `opslevel_id`, `pagerduty_id`, `service_now_ci_sys_id`, `github_repository_name`, `github_repository_branch`, `gitlab_repository_name`, `gitlab_repository_branch`, `kubernetes_deployment_name`, `alerts_email_enabled`

**Functionalities:** `description`, `color`, `backstage_id`, `cortex_id`, `opsgenie_id`, `opsgenie_team_id`, `opslevel_id`, `pagerduty_id`, `service_now_ci_sys_id`

**Environments:** `description`, `color`, `position`

**Teams:** `description`, `color`, `backstage_id`, `cortex_id`, `opsgenie_id`, `opslevel_id`, `pagerduty_id`, `pagerduty_service_id`, `pagertree_id`, `victor_ops_id`, `service_now_ci_sys_id`, `alerts_email_enabled`

Any field not in these lists is routed to catalog properties on the resource.
