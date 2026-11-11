# Contributing

We welcome contributions! Here's how to get started.

## Development setup

```bash
# Clone
git clone https://github.com/rootlyhq/rootly-catalog-sync.git
cd rootly-catalog-sync

# Install dependencies
go mod download

# Build
go build ./cmd/rootly-catalog-sync

# Test
go test ./...

# Lint
golangci-lint run ./...
```

## Making changes

1. Fork the repository
2. Create a feature branch: `git checkout -b my-feature`
3. Make your changes
4. Run tests: `go test ./...`
5. Run lint: `golangci-lint run ./...`
6. Commit with a [Conventional Commit](#commit-messages-and-pr-titles) message
7. Push and open a Pull Request with a Conventional Commit title

## Commit messages and PR titles

Use `<type>(<optional scope>): <description>` for commit messages and PR titles:

```text
feat(source): add a new catalog source [PRF-3498]
fix(backstage): normalize entity references [TER-227]
docs: clarify sync configuration
chore(deps): update dependencies
```

Use `feat` for new features and `fix` for bug fixes. Mark breaking changes with
`!` before the colon, for example `feat!: remove a deprecated configuration option`.
Keep Linear issue IDs at the end of the title and link the ticket in the PR
description; the title must start with the Conventional Commit type.

Release Please uses commit messages to generate release notes and version bumps.
When squash-merging, ensure the final commit title follows this format too.

## Code structure

See [CLAUDE.md](CLAUDE.md) for architecture overview.

## Testing against the Rootly API

```bash
export ROOTLY_API_KEY=rootly_...
rootly-catalog-sync doctor    # verify connectivity
rootly-catalog-sync plan      # preview changes (safe, read-only)
```

## Releasing

Releases are automated via [GoReleaser](.goreleaser.yaml) and triggered by pushing a tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

This builds multi-arch binaries, Docker images, and updates the Homebrew tap.
