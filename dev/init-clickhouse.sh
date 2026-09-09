#!/bin/bash
set -euo pipefail

echo "Initializing ClickHouse schemas..."

# Execute SQL files in numerical order
for sql_file in /opt/clickhouse-schemas/*.sql; do
        echo "Executing: $sql_file"
        if ! clickhouse-client --host localhost --user "$CLICKHOUSE_USER" --password "$CLICKHOUSE_PASSWORD" --queries-file "$sql_file"; then
            echo "Error executing $sql_file - stopping initialization"
            exit 1
        fi
done

echo "ClickHouse schema initialization complete!"
