# ClickHouse Schema Management with Atlas

## Overview

We use [Atlas](https://atlasgo.io/) to manage ClickHouse schema migrations. Atlas provides:
- **Declarative schema management**: Define your desired schema in SQL files
- **Automatic migration generation**: Atlas compares current vs desired state and generates migration files
- **Version control**: All schema changes are tracked as migration files
- **Safe deployments**: Migrations run automatically in both local and production environments

## Workflow

### 1. Edit Schema Files

Schema files are located in `pkg/clickhouse/schema/`. Each file represents a table, view, or other database object:

```
pkg/clickhouse/schema/
├── 001_key_verifications_raw_v2.sql
├── 011_ratelimits_last_used_v2.sql
├── 022_keys_last_used_v2.sql
└── ...
```

**Naming Convention**: Use sequential numbering (001, 002, etc.) to maintain order. The number doesn't affect migration order - it's just for organization.

**Example Schema File**:
```sql
-- 022_keys_last_used_v2.sql
CREATE TABLE IF NOT EXISTS default.keys_last_used_v2
(
    workspace_id String,
    key_id String,
    time Int64 CODEC(Delta, LZ4)
)
ENGINE = ReplacingMergeTree(time)
ORDER BY (workspace_id, key_id, time)
TTL toDateTime(fromUnixTimestamp64Milli(time)) + INTERVAL 90 DAY DELETE;
```

### 2. Generate Migration Files

After editing schema files, generate a migration using Atlas:

```bash
cd pkg/clickhouse

# Make sure Docker is running (Atlas uses it to spin up a temporary ClickHouse instance)
docker ps

# Generate migration
atlas migrate diff \
  --dir "file://migrations" \
  --to "file://schema" \
  --dev-url "docker://clickhouse/latest"
```

**What this does**:
1. Atlas spins up a temporary ClickHouse container
2. Applies all existing migrations to get the current state
3. Compares current state with your schema files
4. Generates a new migration file with the differences
5. Cleans up the temporary container

**Output**: A new migration file in `pkg/clickhouse/migrations/` with a timestamp:
```
migrations/
├── 20240115120000_initial.sql
├── 20240120150000_add_ratelimits.sql
└── 20240126103000_add_keys_last_used.sql  # New file
```

### 3. Review the Migration

Always review the generated migration file before committing:

```bash
cat migrations/20240126103000_add_keys_last_used.sql
```

Check that:
- The SQL statements are correct
- No unexpected changes were included
- The migration is idempotent (safe to run multiple times)

### 4. Commit and Deploy

```bash
git add pkg/clickhouse/schema/022_keys_last_used_v2.sql
git add pkg/clickhouse/migrations/20240126103000_add_keys_last_used.sql
git commit -m "Add keys_last_used_v2 materialized view"
```

**Deployment**: Migrations run automatically when:
- **Local development**: When you start services with `make up` or `make dev`
- **Production**: During deployment via CI/CD pipeline

## Common Scenarios

### Adding a New Table

1. Create a new schema file: `pkg/clickhouse/schema/023_new_table.sql`
2. Define the table structure
3. Run `atlas migrate diff`
4. Review and commit the generated migration

### Modifying an Existing Table

1. Edit the schema file (e.g., add a column to `022_keys_last_used_v2.sql`)
2. Run `atlas migrate diff`
3. Atlas generates an `ALTER TABLE` migration
4. Review and commit

### Creating a Materialized View

1. Create schema file with both target table and materialized view:
```sql
-- Target table
CREATE TABLE IF NOT EXISTS default.my_view_v2 (...) ENGINE = ...;

-- Materialized view that populates it
CREATE MATERIALIZED VIEW IF NOT EXISTS default.my_view_mv_v2
TO default.my_view_v2
AS SELECT ... FROM source_table;
```
2. Run `atlas migrate diff`
3. Review and commit

### Dropping a Table or View

1. Delete or comment out the schema file
2. Run `atlas migrate diff`
3. Atlas generates a `DROP TABLE` migration
4. Review carefully before committing

## Troubleshooting

### "Docker is not running"

Atlas needs Docker to create a temporary ClickHouse instance:
```bash
# Check if Docker is running
docker ps

# Start Docker Desktop or Docker daemon
# Then retry the atlas command
```

### "No schema changes detected"

This means your schema files match the current migration state. Either:
- You forgot to save your schema file changes
- The changes were already migrated
- You're in the wrong directory (must be in `pkg/clickhouse`)

### Migration Conflicts

If multiple people create migrations simultaneously:
1. Pull latest changes: `git pull`
2. Regenerate your migration: `atlas migrate diff`
3. Atlas will create a new migration file with a later timestamp

### Testing Migrations Locally

To test migrations without affecting your local database:

```bash
# Start fresh ClickHouse instance
docker run -d --name clickhouse-test -p 9001:9000 clickhouse/clickhouse-server

# Apply migrations manually
clickhouse-client --host localhost --port 9001 < migrations/20240126103000_add_keys_last_used.sql

# Verify
clickhouse-client --host localhost --port 9001 --query "SHOW TABLES"

# Clean up
docker rm -f clickhouse-test
```

## Best Practices

1. **Always review generated migrations** - Atlas is smart but not perfect
2. **Use IF NOT EXISTS** - Makes migrations idempotent and safe to re-run
3. **Test locally first** - Run `make up` to verify migrations work
4. **Keep schema files in sync** - Schema files should always reflect the desired state
5. **Sequential numbering** - Use next available number for new schema files
6. **Descriptive names** - Name schema files after what they contain (e.g., `022_keys_last_used_v2.sql`)
7. **One concept per file** - Keep related tables/views together, but separate unrelated objects

## Atlas Installation

If you don't have Atlas installed:

```bash
# macOS
brew install ariga/tap/atlas

# Linux
curl -sSf https://atlasgo.sh | sh

# Or download from https://github.com/ariga/atlas/releases
```

Verify installation:
```bash
atlas version
```

## Additional Resources

- [Atlas Documentation](https://atlasgo.io/docs)
- [Atlas ClickHouse Support](https://atlasgo.io/docs/guides/clickhouse)
- [ClickHouse Schema Design](https://clickhouse.com/docs/en/sql-reference/statements/create/table)
