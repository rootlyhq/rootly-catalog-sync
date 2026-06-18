# Simple example: local YAML files

This example syncs a hand-maintained YAML file of services into Rootly's catalog.

## File structure

```
docs/examples/simple/
├── README.md                      # this file
├── rootly-catalog-sync.yaml       # config
└── catalog/
    └── services.yaml              # service data
```

## Walkthrough

### 1. Review the data file

`catalog/services.yaml` contains your services:

```yaml
- id: payments-api
  name: Payments API
  owner: team-payments
  tier: "1"
  description: Handles payment processing

- id: auth-service
  name: Auth Service
  owner: team-platform
  tier: "1"
  description: Authentication and authorization

- id: notification-service
  name: Notification Service
  owner: team-comms
  tier: "2"
  description: Email and push notifications
```

### 2. Review the config

`rootly-catalog-sync.yaml` tells the tool where to find data and how to map it:

```yaml
version: 1
sync_id: simple-example
pipelines:
  - sources:
      - local:
          files: ["catalog/*.yaml"]
    outputs:
      - catalog: "Services"
        external_id: "{{ .id }}"
        name: "{{ .name }}"
        fields:
          owner: "{{ .owner }}"
          tier: "{{ .tier }}"
          description: "{{ .description }}"
```

### 3. Set your API key

```bash
export ROOTLY_API_KEY=rootly_...
```

### 4. Validate the config

```bash
rootly-catalog-sync validate --config=docs/examples/simple/rootly-catalog-sync.yaml
```

Expected output:
```
Config is valid.
```

### 5. Inspect sources

```bash
rootly-catalog-sync sources inspect --config=docs/examples/simple/rootly-catalog-sync.yaml
```

Expected output:
```
Source: local (catalog/*.yaml)
  3 entries loaded

  [0] id=payments-api name=Payments API owner=team-payments tier=1
  [1] id=auth-service name=Auth Service owner=team-platform tier=1
  [2] id=notification-service name=Notification Service owner=team-comms tier=2
```

### 6. Plan (dry run)

```bash
rootly-catalog-sync plan --dry-run --config=docs/examples/simple/rootly-catalog-sync.yaml
```

Expected output:
```
Catalog: Services
  + create  payments-api           (Payments API)
  + create  auth-service           (Auth Service)
  + create  notification-service   (Notification Service)

Summary: 3 to create, 0 to update, 0 to delete, 0 unchanged
```

### 7. Plan + apply

```bash
rootly-catalog-sync sync --config=docs/examples/simple/rootly-catalog-sync.yaml
```

Expected output:
```
Catalog: Services
  + create  payments-api           (Payments API)
  + create  auth-service           (Auth Service)
  + create  notification-service   (Notification Service)

Applying...
  Created 3 entities in catalog "Services"

Done. 3 created, 0 updated, 0 deleted.
```

### 8. Check status (drift detection)

```bash
rootly-catalog-sync status --config=docs/examples/simple/rootly-catalog-sync.yaml
```

Expected output:
```
Catalog: Services — 3 entities, 0 drift
```

## Next steps

- Add more services to `catalog/services.yaml` and re-run `sync`.
- Add `--allow-prune` to delete entries removed from the YAML.
- Set up CI to run `sync` on merge and `plan --dry-run` on PRs.
- See the [backstage example](../backstage/) or [github example](../github/) for remote sources.
