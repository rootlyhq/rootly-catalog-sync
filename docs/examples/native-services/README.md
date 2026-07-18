# Native Services Example

Sync services directly to Rootly's native Services resource (not custom catalog entities).

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

## Config

```yaml
# rootly-catalog-sync.yaml
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

## Usage

```bash
export ROOTLY_API_KEY=rootly_...

# Preview
rootly-catalog-sync plan

# Apply
rootly-catalog-sync sync

# Check
rootly-catalog-sync status
```

## Notes

- `type: service` targets the native Services endpoint (`/v1/services/bulk_upsert`)
- Known attributes (`description`, `pagerduty_id`, `color`, etc.) are set directly on the service
- Any other fields in `fields:` map to catalog properties on the service
- No `catalog` field needed — native resources don't belong to a custom catalog
