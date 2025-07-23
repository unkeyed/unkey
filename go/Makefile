.PHONY: install fmt test-unit test-full build generate lint pull up down generate-docs

DOCS_DIR = ../apps/engineering/content/docs/architecture/services

install:
	go mod tidy

fmt: lint
	go fmt ./...
	golangci-lint run 

pull:
	@docker compose -f ../deployment/docker-compose.yaml pull 

up:
	@docker compose -f ../deployment/docker-compose.yaml up -d mysql redis clickhouse s3 otel
	@echo "Starting ClickHouse migrations (will retry if ClickHouse isn't ready)..."
	@for i in {1..10}; do \
		echo "Migration attempt $i..."; \
		if docker compose -f ../deployment/docker-compose.yaml run --rm clickhouse_migrator; then \
			echo "Migrations completed successfully!"; \
			break; \
		else \
			echo "Migration failed, retrying in 5 seconds..."; \
			sleep 5; \
		fi; \
	done

down:
	@docker compose -f ../deployment/docker-compose.yaml down

test-full: export INTEGRATION_TEST=true
test-full: export SIMULATION_TEST=false
test-full: up
	@echo "Running full tests... this can take more than 30min... run 'make test-unit' for faster tests"
	@go test -failfast -timeout=60m -shuffle=on -v -json ./... | tparse -all -progress -smallscreen
test-unit: up
	@go test -json -race -failfast -timeout=30m ./... | tparse -all -progress -smallscreen

# Updated Makefile
build: generate-docs
build-binary:
	go build -o unkey ./main.go

generate:
	buf generate
	go generate ./...
generate-docs: build-binary
	@echo "Starting automated documentation generation..."
	@echo "DOCS_DIR: $(DOCS_DIR)"
	@total=0; success=0; \
	for cmd in deploy run healthcheck quotacheck version; do \
		echo ""; \
		echo "=== Processing $$cmd ==="; \
		mkdir -p "$(DOCS_DIR)/$$cmd"; \
		total=$$((total + 1)); \
		if ./unkey $$cmd mdx > "$(DOCS_DIR)/$$cmd/index.mdx" 2>/dev/null; then \
			echo "✓ Generated $(DOCS_DIR)/$$cmd/index.mdx"; \
			success=$$((success + 1)); \
		else \
			echo "✗ Failed to generate $$cmd"; \
			continue; \
		fi; \
		\
		help_output=$$(./unkey $$cmd --help 2>/dev/null); \
		if echo "$$help_output" | grep -q "COMMANDS:"; then \
			subcmds=$$(echo "$$help_output" | awk '/COMMANDS:/{flag=1; next} /^$$/{flag=0} flag && /^   [a-zA-Z]/ {gsub(/,.*/, "", $$1); print $$1}' | grep -v "help"); \
			subcount=$$(echo "$$subcmds" | wc -w); \
			echo "  Found $$subcount subcommands: $$subcmds"; \
			for subcmd in $$subcmds; do \
				if [ "$$subcmd" != "" ]; then \
					mkdir -p "$(DOCS_DIR)/$$cmd/$$subcmd"; \
					total=$$((total + 1)); \
					if ./unkey $$cmd $$subcmd mdx > "$(DOCS_DIR)/$$cmd/$$subcmd/index.mdx" 2>/dev/null; then \
						echo "  ✓ Generated $(DOCS_DIR)/$$cmd/$$subcmd/index.mdx"; \
						success=$$((success + 1)); \
					else \
						echo "  ✗ Failed to generate $$cmd/$$subcmd"; \
					fi; \
				fi; \
			done; \
		else \
			echo "  No subcommands found"; \
		fi; \
	done; \
	echo ""; \
	echo "Summary: $$success/$$total files generated successfully"

# Dry run version to see what would be generated
generate-docs-dry-run: build-binary
	@echo "Dry run docs generation..."
	@for cmd in deploy run healthcheck quotacheck version; do \
		echo "Will generate: $(DOCS_DIR)/$$cmd/index.mdx"; \
		help_output=$$(./unkey $$cmd --help 2>/dev/null); \
		if echo "$$help_output" | grep -q "COMMANDS:"; then \
			subcmds=$$(echo "$$help_output" | awk '/COMMANDS:/{flag=1; next} /^$$/{flag=0} flag && /^   [a-zA-Z]/ {gsub(/,.*/, "", $$1); print $$1}' | grep -v "help"); \
			for subcmd in $$subcmds; do \
				if [ "$$subcmd" != "" ]; then \
					echo "  Would generate: $(DOCS_DIR)/$$cmd/$$subcmd/index.mdx"; \
				fi; \
			done; \
		fi; \
	done
