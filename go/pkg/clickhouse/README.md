# Changing schemas

To change a schema, just edit a table in the schema folder.
Afterwards generate a migration file using atlas.
Make sure docker is running, so atla can use it to generate migrations.

```bash
cd go/pkg/clickhouse
atlas migrate diff \
	--dir "file://migrations" \
	--to "file://schema" \
	--dev-url "docker://clickhouse/latest"
```
