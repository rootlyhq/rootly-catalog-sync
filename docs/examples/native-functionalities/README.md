# Native Functionalities Example

Sync functionalities directly to Rootly's native Functionalities resource.

## Config

```yaml
version: 1
sync_id: native-functionalities
pipelines:
  - sources:
      - local:
          files: ["functionalities.yaml"]
    outputs:
      - type: functionality
        external_id: "{{ .id }}"
        name: "{{ .name }}"
        fields:
          description: "{{ .description }}"
```

## Data file

```yaml
# functionalities.yaml
- id: checkout-flow
  name: Checkout Flow
  description: End-to-end purchase and payment flow

- id: user-auth
  name: User Authentication
  description: Login, SSO, and session management

- id: search
  name: Search & Discovery
  description: Product search, filters, and recommendations
```

## Usage

```bash
export ROOTLY_API_KEY=rootly_...
rootly-catalog-sync sync
```

## Supported fields

`description`, `color`, `backstage_id`, `cortex_id`, `opsgenie_id`, `opsgenie_team_id`, `opslevel_id`, `pagerduty_id`, `service_now_ci_sys_id`, `notify_emails`
