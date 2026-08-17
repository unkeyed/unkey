.PHONY: build dev lint

build:
	@docker build -t unkey-engineering-docs .

dev: build
	@docker run --rm -it \
	  -p 3000:3000 \
	  -v $(CURDIR):/app \
	  -v /app/node_modules \
	  unkey-engineering-docs

lint: build
	@docker run --rm \
	  -v $(CURDIR):/app \
	  -v /app/node_modules \
	  unkey-engineering-docs npm run check
	@docker run --rm \
	  -v $(CURDIR):/app \
	  -v /app/node_modules \
	  unkey-engineering-docs npm run validate
