# Native Teams Example

Sync teams directly to Rootly's native Teams (Groups) resource.

**Note:** At least one team must always remain — the sync tool will refuse to delete all teams.

## Config

```yaml
version: 1
sync_id: native-teams
pipelines:
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
          pagerduty_id: "{{ .pagerduty_id }}"
```

## Data file

```yaml
# teams.yaml
- id: team-platform
  name: Platform Engineering
  description: Infrastructure, CI/CD, and developer tools
  color: "#7C3AED"
  pagerduty_id: PXXXXXX

- id: team-payments
  name: Payments
  description: Payment processing and billing
  color: "#10B981"

- id: team-security
  name: Security
  description: Application and infrastructure security
  color: "#EF4444"
```

## Usage

```bash
export ROOTLY_API_KEY=rootly_...
rootly-catalog-sync sync
```

## Supported fields

`description`, `color`, `position`, `backstage_id`, `cortex_id`, `opsgenie_id`, `opslevel_id`, `pagerduty_id`, `pagerduty_service_id`, `pagertree_id`, `victor_ops_id`, `service_now_ci_sys_id`, `alerts_email_enabled`, `notify_emails`
