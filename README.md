# plugin-morphediff-psql

Generates PostgreSQL migration SQL files from MorpheDiff change documents (`KA:MD1:YAML1`).

## What it generates

Given a MorpheDiff YAML document describing schema changes between two Morphe versions,
this plugin produces individual `.sql` migration files:

| Change type     | Operations supported                       | SQL output                                         |
|-----------------|--------------------------------------------|----------------------------------------------------|
| **Model**       | add, remove, modify, rename, deprecate     | `CREATE TABLE`, `DROP TABLE`, `ALTER TABLE`, `COMMENT ON TABLE` |
| **Field**       | add, remove, modify, rename                | `ALTER TABLE ... ADD/DROP/ALTER COLUMN`, `RENAME COLUMN` |
| **Enum**        | add, remove, modify, rename, deprecate     | `CREATE TYPE`, `DROP TYPE`, `ALTER TYPE`            |
| **Relationship**| add, remove, modify                        | `ALTER TABLE ... ADD COLUMN/CONSTRAINT`, `DROP CONSTRAINT` |
| **Entity**      | add, remove, modify, rename, deprecate     | Stub migrations (entities map to tables similarly to models) |
| **Structure**   | add, remove, modify, rename, deprecate     | Comment-only placeholders (structures are value objects, no dedicated table) |

Polymorphic relationships (`ForOnePoly`, `ForManyPoly`) are fully supported with
`_type`/`_id` columns, composite indexes, and optional `CHECK` constraints.

## Input / output

| Direction | Format             | Store suggestion |
|-----------|--------------------|------------------|
| Input     | `KA:MD1:YAML1`     | `KA_MORPHE_DIFF` |
| Output    | `KA:PSQL:MIGRATION1` | `KA_MIGRATIONS` |

## Configuration

| Key              | Type    | Default    | Description |
|------------------|---------|------------|-------------|
| `schemaName`     | string  | `"public"` | PostgreSQL schema for generated statements |
| `useBigSerial`   | boolean | `false`    | Use `BIGSERIAL` instead of `SERIAL` for auto-increment columns |
| `timestampPrefix`| boolean | `true`     | Prefix migration filenames with `YYYYMMDDHHMMSS_NNN_` |

## Pipeline context

This plugin is typically used in a **diff pipeline** after
[`plugin-morphe-git-morphediff`](../plugin-morphe-git-morphediff) generates the
MorpheDiff document from two Git revisions:

```yaml
diff:
  stages:
    - name: "morphe-diff"
      steps:
        - "plugin: @kalo-build/plugin-morphe-git-morphediff"
    - name: "psql-migrations"
      steps:
        - "plugin: @kalo-build/plugin-morphediff-psql"
```

## Project structure

```
plugin-morphediff-psql/
├── cmd/plugin/             # WASM entry point
├── pkg/
│   ├── compile/            # Migration generators (models, enums, fields, relationships, entities, structures)
│   ├── diffdef/            # MorpheDiff YAML type definitions
│   └── formatdef/          # ContentBuilder helper + naming utilities
├── testdata/
│   ├── morphe-diff.yaml    # Sample input
│   └── expected/           # Ground-truth SQL for integration tests
├── dist/                   # WASM output
└── plugin.yaml             # Kalo plugin manifest
```

## Building

```bash
# Native binary
go build ./cmd/plugin

# WASM (for Kalo CLI)
GOOS=wasip1 GOARCH=wasm go build -o dist/plugin.wasm cmd/plugin/main.go
```

## Testing

```bash
go test ./...
```

## License

See [LICENSE](LICENSE).
