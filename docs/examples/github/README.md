# GitHub example: sync from GitHub org repositories

This example discovers `catalog.yaml` files across all repositories in a GitHub organization and syncs them into Rootly.

## File structure

```
docs/examples/github/
├── README.md                      # this file
└── rootly-catalog-sync.yaml       # config
```

## Prerequisites

- A GitHub personal access token or fine-grained token stored in `GITHUB_TOKEN`.
  - Classic tokens: `repo` scope (private repos) or `public_repo` (public only).
  - Fine-grained tokens: **Contents: Read** on target repositories.

## Config

`rootly-catalog-sync.yaml` scans repos for catalog data files:

```yaml
version: 2

sync:
  - from:
      github:
        token: "$(GITHUB_TOKEN)"
        owner: acme
        files: ["catalog.yaml", "**/catalog.yaml"]
        archived: false
    to: Services
    map:
      external_id: "{{ .id }}"
      name: "{{ .name }}"
      owner: "{{ .owner }}"
      tier: "{{ default .tier \"3\" }}"
      description: "{{ default .description \"\" }}"
```

<details>
<summary>v1 format (still supported)</summary>

```yaml
version: 1
sync_id: github-services
pipelines:
  - sources:
      - github:
          token: "$(GITHUB_TOKEN)"
          owner: acme
          files: ["catalog.yaml", "**/catalog.yaml"]
          archived: false
    outputs:
      - catalog: "Services"
        external_id: "{{ .id }}"
        name: "{{ .name }}"
        fields:
          owner:
            value: "{{ .owner }}"
          tier:
            value: "{{ default .tier \"3\" }}"
          description:
            value: "{{ default .description \"\" }}"
```
</details>

## How it works

1. The `github` source lists all repositories in the `acme` organization.
2. Archived and forked repositories are skipped by default.
3. For each repo, it fetches the full file tree and matches against `files` patterns.
4. Matching files are fetched, parsed as YAML, and emitted as source entries.
5. Templates map entry fields to catalog entity attributes.

## File patterns

The `files` field supports both single-level globs and doublestar recursive patterns:

| Pattern | Matches |
|---------|---------|
| `catalog.yaml` | `catalog.yaml` in the repo root |
| `*.yaml` | Any `.yaml` file in the repo root |
| `**/catalog.yaml` | `catalog.yaml` at any depth |
| `services/**/*.yaml` | Any `.yaml` under `services/` at any depth |
| `config/catalog-*.yaml` | `config/catalog-prod.yaml`, `config/catalog-staging.yaml`, etc. |

## Specific repos

To sync only specific repositories instead of the entire org:

```yaml
from:
  github:
    token: "$(GITHUB_TOKEN)"
    owner: acme
    repos: ["payments", "auth", "gateway"]
    files: ["catalog.yaml"]
```

## Custom branch

By default, the repo's default branch is used. To pin to a specific ref:

```yaml
from:
  github:
    token: "$(GITHUB_TOKEN)"
    owner: acme
    files: ["catalog.yaml"]
    ref: main
```

## Including archived repos

```yaml
from:
  github:
    token: "$(GITHUB_TOKEN)"
    owner: acme
    files: ["catalog.yaml"]
    archived: true
```

## Usage

```bash
export ROOTLY_API_KEY=rootly_...
export GITHUB_TOKEN=ghp_...

# Preview
rootly-catalog-sync plan --dry-run --config=docs/examples/github/rootly-catalog-sync.yaml

# Apply
rootly-catalog-sync sync --config=docs/examples/github/rootly-catalog-sync.yaml
```

## Expected catalog.yaml in each repo

Each repository should contain a `catalog.yaml` (or matching file) with entries like:

```yaml
- id: payments-api
  name: Payments API
  owner: team-payments
  tier: "1"
  description: Core payment processing service
```

Or a single entry:

```yaml
id: payments-api
name: Payments API
owner: team-payments
tier: "1"
```

Both single-object and array formats are supported.
