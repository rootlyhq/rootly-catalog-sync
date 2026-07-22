# Config Formats

rootly-catalog-sync supports three config file formats, detected by extension: `.yaml`/`.yml`, `.jsonnet`, and `.hcl`. All three support both v1 and v2 config schemas.

Each section below shows the same logical example — syncing services from local YAML files — so you can compare formats directly.

---

## YAML (recommended)

YAML is the default and most common format. Use `.yaml` or `.yml` extension.

### Catalog entity example

```yaml
version: 2

sync:
  - from:
      local:
        files: ["catalog/*.yaml"]
    to: Services
    map:
      external_id: "{{ .id }}"
      name: "{{ .name }}"
      description: "{{ .description }}"
      owner: "{{ .owner }}"
```

### Native resource example

Use a lowercase type name (`service`, `team`, `environment`, `functionality`) instead of a catalog name:

```yaml
version: 2

sync:
  - from:
      local:
        files: ["services.yaml"]
    to: service
    map:
      external_id: "{{ .id }}"
      name: "{{ .name }}"
      description: "{{ .description }}"
```

### Reference field example

Reference fields point to entries in another catalog. Define the reference data first, then use `reference:` to link to it:

```yaml
version: 2

sync:
  - from:
      local:
        files: ["tiers.yml"]
    to: Tiers
    map:
      external_id: "{{ .id }}"
      name: "{{ .name }}"

  - from:
      local:
        files: ["services.yaml"]
    to: service
    map:
      external_id: "{{ .id }}"
      name: "{{ .name }}"
      description: "{{ .description }}"
      tier:
        value: "{{ .tier }}"
        reference: Tiers
```

Sync entries are processed in order — reference catalogs must come before the resources that use them.

<details>
<summary>v1 format (still supported)</summary>

```yaml
version: 1
sync_id: services
pipelines:
  - sources:
      - local:
          files: ["catalog/*.yaml"]
    outputs:
      - catalog: "Services"
        external_id: "{{ .id }}"
        name: "{{ .name }}"
        fields:
          description:
            value: "{{ .description }}"
          owner:
            value: "{{ .owner }}"
```
</details>

---

## Jsonnet

Jsonnet adds variables, functions, and imports for DRY configs. Use `.jsonnet` extension.

> **Note:** `local` is a reserved keyword in Jsonnet and must be quoted as `"local"` when used as a source key.

### Catalog entity example

```jsonnet
// rootly-catalog-sync.jsonnet
{
  version: 2,
  sync: [
    {
      from: {
        "local": {
          files: ["catalog/*.yaml"],
        },
      },
      to: "Services",
      map: {
        external_id: "{{ .id }}",
        name: "{{ .name }}",
        description: "{{ .description }}",
        owner: "{{ .owner }}",
      },
    },
  ],
}
```

### Native resource example

```jsonnet
{
  version: 2,
  sync: [
    {
      from: {
        "local": {
          files: ["services.yaml"],
        },
      },
      to: "service",
      map: {
        external_id: "{{ .id }}",
        name: "{{ .name }}",
        description: "{{ .description }}",
      },
    },
  ],
}
```

### Reference field example

```jsonnet
{
  version: 2,
  sync: [
    {
      from: {
        "local": {
          files: ["tiers.yml"],
        },
      },
      to: "Tiers",
      map: {
        external_id: "{{ .id }}",
        name: "{{ .name }}",
      },
    },
    {
      from: {
        "local": {
          files: ["services.yaml"],
        },
      },
      to: "service",
      map: {
        external_id: "{{ .id }}",
        name: "{{ .name }}",
        description: "{{ .description }}",
        tier: {value: "{{ .tier }}", reference: "Tiers"},
      },
    },
  ],
}
```

```bash
rootly-catalog-sync sync --config=rootly-catalog-sync.jsonnet
```

<details>
<summary>v1 format (still supported)</summary>

```jsonnet
{
  version: 1,
  sync_id: "services",
  pipelines: [
    {
      sources: [
        {
          "local": {
            files: ["catalog/*.yaml"],
          },
        },
      ],
      outputs: [
        {
          catalog: "Services",
          external_id: "{{ .id }}",
          name: "{{ .name }}",
          fields: {
            description: {value: "{{ .description }}"},
            owner: {value: "{{ .owner }}"},
          },
        },
      ],
    },
  ],
}
```
</details>

---

## HCL

HCL provides Terraform-style syntax with blocks. Use `.hcl` extension.

### Catalog entity example

```hcl
# rootly-catalog-sync.hcl
version = 2

sync {
  from {
    local {
      files = ["catalog/*.yaml"]
    }
  }
  to  = "Services"
  map = {
    external_id = "{{ .id }}"
    name        = "{{ .name }}"
    description = "{{ .description }}"
    owner       = "{{ .owner }}"
  }
}
```

### Native resource example

```hcl
version = 2

sync {
  from {
    local {
      files = ["services.yaml"]
    }
  }
  to  = "service"
  map = {
    external_id = "{{ .id }}"
    name        = "{{ .name }}"
    description = "{{ .description }}"
  }
}
```

### Reference field example

```hcl
version = 2

sync {
  from {
    local {
      files = ["tiers.yml"]
    }
  }
  to  = "Tiers"
  map = {
    external_id = "{{ .id }}"
    name        = "{{ .name }}"
  }
}

sync {
  from {
    local {
      files = ["services.yaml"]
    }
  }
  to  = "service"
  map = {
    external_id = "{{ .id }}"
    name        = "{{ .name }}"
    description = "{{ .description }}"
    tier        = { value = "{{ .tier }}", reference = "Tiers" }
  }
}
```

```bash
rootly-catalog-sync sync --config=rootly-catalog-sync.hcl
```

<details>
<summary>v1 format (still supported)</summary>

```hcl
version = 1
sync_id = "services"

pipeline {
  source {
    local {
      files = ["catalog/*.yaml"]
    }
  }
  output {
    catalog     = "Services"
    external_id = "{{ .id }}"
    name        = "{{ .name }}"
    fields = {
      description = {
        value = "{{ .description }}"
      }
      owner = {
        value = "{{ .owner }}"
      }
    }
  }
}
```
</details>
