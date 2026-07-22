# Simple example: local YAML files

Sync a hand-maintained YAML file of services into Rootly's catalog.

## File structure

```
catalog/services.yaml          — service data
rootly-catalog-sync.yaml       — config
```

## Data file

```yaml
# catalog/services.yaml
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

## Config

```yaml
# rootly-catalog-sync.yaml
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
          owner:
            value: "{{ .owner }}"
          tier:
            value: "{{ .tier }}"
          description:
            value: "{{ .description }}"
```

## Usage

```bash
export ROOTLY_API_KEY=rootly_...

rootly-catalog-sync validate     # check config
rootly-catalog-sync doctor       # verify auth + connectivity
rootly-catalog-sync plan         # preview changes
rootly-catalog-sync sync         # apply
rootly-catalog-sync status       # verify in sync
```

## Next steps

- Add more services and re-run `sync`
- Add `--allow-prune` to delete entries removed from YAML
- Set up CI to run `sync` on merge and `plan --dry-run` on PRs
- See [backstage](../backstage/) or [github](../github/) for remote sources
