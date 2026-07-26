BINARY  := tidefetch
VERSION := 0.2.0
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build backend web site site-dev docs-check run serve test vet fmt install docker clean

## build: web UI + binary with embedded assets
build: web backend

## backend: compile the Go binary only (uses last built web/dist)
backend:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/$(BINARY)

## web: compile the Svelte frontend into web/dist
web:
	cd web && npm install --no-fund --no-audit && npm run build

## site: compile the standalone product/showcase site
site:
	cd site && npm install --no-fund --no-audit && npm run check && npm run build

## site-dev: run the product site development server
site-dev:
	cd site && npm run dev

## docs-check: verify every relative Markdown link resolves
docs-check:
	node scripts/check-doc-links.mjs

run: backend
	./$(BINARY)

serve: backend
	./$(BINARY) serve

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .
	cd web && npx svelte-check --threshold error 2>/dev/null || true

install: build
	go install -trimpath -ldflags "$(LDFLAGS)" ./cmd/$(BINARY)

docker:
	docker build -f packaging/docker/Dockerfile -t tidefetch:$(VERSION) .

clean:
	rm -f $(BINARY)
	rm -rf web/node_modules site/node_modules site/dist
