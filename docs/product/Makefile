.PHONY: build dev lint

build:
	@docker build -t unkey-docs .

dev: build
	@docker run --rm -it \
  	-p 3000:3000 \
    -v $(CURDIR):/app \
    unkey-docs

lint: build
	@docker run --rm \
	  -v $(CURDIR):/app \
	  unkey-docs sh -c "mintlify broken-links && mintlify validate && mintlify a11y"

