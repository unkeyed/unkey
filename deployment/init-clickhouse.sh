#!/bin/bash
set -e

echo "Initializing ClickHouse schemas..."

# Execute SQL files in order from our schemas directory
for db_dir in /opt/clickhouse-schemas/*/; do
    if [ -d "$db_dir" ]; then
        echo "Processing database directory: $db_dir"
        
        # Execute SQL files in numerical order
        for sql_file in "$db_dir"*.sql; do
            if [ -f "$sql_file" ] && [[ "$sql_file" == *.sql ]]; then
                echo "Executing: $sql_file"
                
                if ! clickhouse-client --host localhost --user "$CLICKHOUSE_ADMIN_USER" --password "$CLICKHOUSE_ADMIN_PASSWORD" --queries-file "$sql_file"; then
                    echo "Error executing $sql_file - stopping initialization"
                    exit 1
                fi
            fi
        done
    fi
done

echo "ClickHouse schema initialization complete!"