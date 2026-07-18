# Native Services Example

Sync services directly to Rootly's native Services resource.

## Config

```yaml
version: 1
sync_id: native-services
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
```

## Data file

```yaml
# services.yaml
- id: payments-api
  name: Payments API
  description: Handles all payment processing
  pagerduty_id: P123ABC

- id: auth-service
  name: Auth Service
  description: Authentication and authorization
```

## Usage

```bash
export ROOTLY_API_KEY=rootly_...
rootly-catalog-sync plan    # preview
rootly-catalog-sync sync    # apply
rootly-catalog-sync status  # verify
```

## Supported fields

`description`, `color`, `backstage_id`, `cortex_id`, `opsgenie_id`, `opsgenie_team_id`, `opslevel_id`, `pagerduty_id`, `service_now_ci_sys_id`, `github_repository_name`, `github_repository_branch`, `gitlab_repository_name`, `gitlab_repository_branch`, `kubernetes_deployment_name`, `alerts_email_enabled`, `notify_emails`
