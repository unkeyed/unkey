

create-migrations:
	atlas migrate diff init \
		--dir "file://schema/migrations" \
		--to "file://schema/old_schema.sql" \
		--dev-url "docker://clickhouse/latest"
