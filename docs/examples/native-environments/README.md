# Native Environments Example

Sync environments directly to Rootly's native Environments resource.

**Note:** At least one environment must always remain — the sync tool will refuse to delete all environments.

## Config

```yaml
version: 1
sync_id: native-environments
pipelines:
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

## Data file

```yaml
# environments.yaml
- id: production
  name: Production
  description: Live customer-facing environment
  color: "#EF4444"

- id: staging
  name: Staging
  description: Pre-production testing environment
  color: "#F59E0B"

- id: development
  name: Development
  description: Local development environment
  color: "#10B981"
```

## Usage

```bash
export ROOTLY_API_KEY=rootly_...
rootly-catalog-sync sync
```

## Supported fields

`description`, `color`, `position`, `notify_emails`
