.PHONY: sync dev lint

sync:
	@bun run scripts/sync-infra-docs.ts

dev: sync
	bunx mintlify@latest dev

lint: sync
	bunx mintlify@latest broken-links
	bunx mintlify@latest validate
	bunx mintlify@latest a11y
