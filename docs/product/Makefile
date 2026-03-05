.PHONY: build dev

build:
	@docker build -t unkey-docs .

dev: build
	@docker run --rm -it \
  	-p 3000:3000 \
    -v /Users/chronark/github/unkeyed/unkey/docs/product:/app \
    unkey-docs

