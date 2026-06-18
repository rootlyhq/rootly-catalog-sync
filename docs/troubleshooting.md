# Troubleshooting

## Quick diagnostics

Run these commands in order to isolate issues:

```bash
# 1. Check environment, auth, and API connectivity
rootly-catalog-sync doctor

# 2. Validate config syntax and schema
rootly-catalog-sync validate --config=rootly-catalog-sync.yaml

# 3. Dump raw entries from sources before mapping
rootly-catalog-sync sources inspect --config=rootly-catalog-sync.yaml

# 4. Preview changes without applying
rootly-catalog-sync plan --dry-run

# 5. Trace a single entry through source -> mapping -> diff
rootly-catalog-sync explain <external_id>
```

---

## Authentication

### `ROOTLY_API_KEY environment variable is required`

**Cause:** The `ROOTLY_API_KEY` env var is not set.

**Fix:**
```bash
export ROOTLY_API_KEY=rootly_...
```

In CI, set it as a secret:
```yaml
env:
  ROOTLY_API_KEY: ${{ secrets.ROOTLY_API_KEY }}
```

### `listing catalogs: unexpected status 401`

**Cause:** The API key is invalid or revoked.

**Fix:**
1. Verify the key starts with `rootly_`.
2. Generate a new key at **Settings > API Keys** in the Rootly dashboard.
3. If using `$(ENV_VAR)` syntax in your config, make sure the referenced env var is set.

### `listing catalogs: unexpected status 404`

**Cause:** The API URL or path is wrong.

**Fix:**
- Check `ROOTLY_API_URL` (default: `https://api.rootly.com`).
- Check `ROOTLY_API_PATH` (default: `/v1`).
- If you are on a dedicated Rootly instance, use `ROOTLY_API_URL=https://api.<your-domain>.rootly.com`.

---

## Config

### `reading config file: open rootly-catalog-sync.yaml: no such file or directory`

**Cause:** Config file not found at the default or specified path.

**Fix:**
- Pass `--config=path/to/config.yaml` explicitly.
- Run `rootly-catalog-sync init` to generate a starter config.

### `parsing config file: yaml: ...`

**Cause:** YAML syntax error (bad indentation, missing colon, tabs instead of spaces, etc.).

**Fix:**
- Run `rootly-catalog-sync validate --config=...` to see the exact line/column.
- Use `rootly-catalog-sync fmt` to auto-fix formatting issues.

### `unsupported config version: 2 (expected 1)`

**Cause:** The `version` field is not `1`.

**Fix:** Set `version: 1` at the top of your config.

### `sync_id is required`

**Cause:** Missing `sync_id` field.

**Fix:** Add a unique `sync_id` to your config:
```yaml
version: 1
sync_id: my-services
```

### `pipeline[0]: at least one source is required`

**Cause:** A pipeline has no `sources` array, or it is empty.

**Fix:** Add at least one source block to each pipeline.

### `pipeline[0].outputs[0]: catalog is required`

**Cause:** Missing `catalog` field in an output block.

**Fix:** Every output must specify a `catalog` name, `external_id`, and `name`:
```yaml
outputs:
  - catalog: "Services"
    external_id: "{{ .id }}"
    name: "{{ .name }}"
```

### `no source type configured`

**Cause:** A source block exists but has no recognized source type key (`local`, `github`, `exec`, etc.).

**Fix:** Add exactly one source type to each source block:
```yaml
sources:
  - local:
      files: ["catalog/*.yaml"]
```

### `multiple source types configured; exactly one is required`

**Cause:** A single source block contains more than one source type.

**Fix:** Split each source type into its own source block:
```yaml
sources:
  - local:
      files: ["catalog/*.yaml"]
  - github:
      owner: acme
      repos: ["payments"]
      files: ["catalog.yaml"]
```

### `evaluating jsonnet: ...`

**Cause:** Jsonnet evaluation failed (syntax error, undefined variable, bad import path).

**Fix:**
- Check Jsonnet syntax at the line mentioned in the error.
- Ensure all `import` paths are relative to the config file location.
- Run `jsonnet rootly-catalog-sync.jsonnet` directly to debug.

### `converting hcl to json: parsing hcl: ...`

**Cause:** HCL parse error (missing brace, bad attribute syntax, etc.).

**Fix:**
- Check HCL syntax at the line/column in the error.
- Ensure block types are `pipeline`, `source`, and `output` (not pluralized).
- Run `hclfmt rootly-catalog-sync.hcl` to auto-format.

---

## Sources

### `loading source local: no files matched pattern "catalog/*.yaml"`

**Cause:** The glob pattern did not match any files on disk.

**Fix:**
- Check that the files exist relative to the config file directory.
- Use `rootly-catalog-sync sources inspect` to see what each source loads.
- Verify the pattern: `*.yaml` matches one directory level, `**/*.yaml` matches recursively.

### `loading source csv: no files matched ...`

**Cause:** Same as above but for CSV source.

**Fix:** Verify the file paths and glob patterns in your `csv.files` config.

### `loading source github: resolving repos: ...`

**Cause:** GitHub API call failed. Common reasons:
- Missing or invalid `GITHUB_TOKEN`.
- Token lacks `repo` scope for private repos.
- Organization name (`owner`) is misspelled.

**Fix:**
- Set a valid `GITHUB_TOKEN` with at least `repo` (private) or `public_repo` (public-only) scope.
- Verify `owner` matches the exact GitHub organization or user name.
- For fine-grained tokens, grant **Contents: Read** on the target repositories.

### `getting repo acme/payments: 404 Not Found`

**Cause:** The repository does not exist, or the token does not have access.

**Fix:**
- Check the repo name for typos.
- Ensure the token has access to the repository.

### Archived repos are skipped

**Cause:** By default, `archived: false` skips archived repositories.

**Fix:** If you need archived repos, set `archived: true` in the GitHub source config:
```yaml
sources:
  - github:
      owner: acme
      archived: true
      files: ["catalog.yaml"]
```

### `warning: repo acme/my-repo has no tree at ref "main", skipping`

**Cause:** The repo is empty or the specified `ref` does not exist.

**Fix:**
- Verify the branch/ref exists in the repository.
- Omit `ref` to use the repo's default branch.

### `loading source exec: ...`

**Cause:** The `exec` command failed (non-zero exit, command not found, timeout).

**Fix:**
- Test the command manually: `bq query --format=json "SELECT ..."`.
- Ensure the command is in `$PATH`.
- Check that stdout is valid JSON (array of objects).

### `loading source backstage: backstage API returned status 403`

**Cause:** Backstage token is invalid or lacks permissions.

**Fix:**
- Verify `BACKSTAGE_TOKEN` is set and valid.
- Check the Backstage service account has read access to entities.

### Empty source safety abort

**Error:**
```
safety check failed: empty source: refusing to delete all 42 live entities -- source returned 0 entries
```

**Cause:** Every source in the pipeline returned zero entries. This is treated as a source failure, not an intentional "delete everything."

**Fix:**
- Use `rootly-catalog-sync sources inspect` to see what each source returns.
- Fix the underlying source issue (permissions, network, empty files).
- If you truly want to delete all entries, remove them from the source data rather than breaking the source.

---

## Mapping

### `entry 0: evaluating external_id: executing template: ... map has no entry for key "id"`

**Cause:** The template references a field (`.id`) that does not exist in the source entry. Templates use `missingkey=error` by default.

**Fix:**
- Run `rootly-catalog-sync sources inspect` to see the actual field names.
- Fix the template to use the correct field name.
- Use `{{ default .id "" }}` if the field is optional.

### `entry 0: evaluating external_id: parsing template: ...`

**Cause:** Go template syntax error (unclosed braces, bad function call, etc.).

**Fix:**
- Check for matching `{{` and `}}`.
- Verify function names: `get`, `default` are built-in; see [templates.md](templates.md).

### `entry 0: external_id evaluated to empty`

**Cause:** The `external_id` template produced an empty string after evaluation.

**Fix:**
- Check that the source field is not null/empty.
- Run `rootly-catalog-sync sources inspect` and look at the specific entry.
- Ensure the template references the correct field.

### `entry 0: name evaluated to empty`

**Cause:** Same as above, but for the `name` field.

**Fix:** Ensure every source entry has a non-empty value for the field used in the `name` template.

---

## Reconcile

### `safety check failed: prune ratio 50% (5/10) exceeds threshold 20%`

**Cause:** More than 20% of live entries would be deleted. This is a safety guard to prevent accidental mass deletion.

**Fix:**
- If the deletions are intentional, raise the threshold:
  ```bash
  rootly-catalog-sync sync --prune-threshold=0.5
  ```
- If not intentional, check why entries are missing from the source.
- Use `rootly-catalog-sync plan --dry-run` to review what would be deleted.

### `plan is stale (re-run 'plan' to get a fresh plan, or use --force to apply anyway)`

**Cause:** The live state in Rootly changed after `plan` was run but before `apply`. Someone may have edited entries in the UI, or another sync ran.

**Fix:**
- Re-run `plan` to get a fresh plan, then `apply` the new plan file.
- If you are confident the plan is still correct: `rootly-catalog-sync apply --force <plan-file>`.

---

## Connectivity

### `after 3 retries: HTTP 429`

**Cause:** Rate-limited by the Rootly API. The client automatically retries with exponential backoff and honors the `Retry-After` header.

**Fix:**
- This usually resolves on its own. If it persists, reduce the number of entities per batch or add a delay between syncs.
- Check if multiple sync jobs are running concurrently against the same organization.

### `after 3 retries: ... connection refused` / `dial tcp: lookup api.rootly.com: no such host`

**Cause:** DNS resolution failed or the API host is unreachable.

**Fix:**
- Check network connectivity: `curl -I https://api.rootly.com/v1/catalogs`.
- Verify `ROOTLY_API_URL` is correct.
- In Docker, ensure the container has network access (not `--network=none`).

### `context deadline exceeded`

**Cause:** API request timed out (default 10 seconds for `doctor`, longer for sync operations).

**Fix:**
- Check network latency to the API.
- If behind a corporate proxy, configure `HTTP_PROXY`/`HTTPS_PROXY`.

---

## TUI

### Terminal too narrow for detail pane

**Cause:** The TUI needs a minimum terminal width to display the side-by-side diff panel.

**Fix:**
- Widen your terminal window to at least 100 columns.
- Or use `plan --dry-run` for a text-based diff that works at any width.

### Alt-screen issues (garbled output after exit)

**Cause:** The TUI uses the alternate screen buffer. If it exits abnormally, your terminal may be in a bad state.

**Fix:**
- Run `reset` or `tput rmcup` to restore the normal screen.
- If running in `tmux` or `screen`, ensure the terminal emulator supports alt-screen.
