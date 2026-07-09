# kitchensink

Stdlib-only HTTP server. Each probe lives in its own subpackage and
demonstrates one platform feature end-to-end — sentinel policies,
deployment routing, env injection, header propagation, and so on.

Probes double as worked examples: an engineer reading one subpackage
should get a complete, copy-pasteable picture of how to integrate with
that feature without chasing imports.

## Constraints

- **Go standard library only.** No third-party dependencies. The only
  allowed local import is `internal/httpx` for trivial response
  plumbing (e.g. `httpx.JSON`) — kept under `internal/` so it stays
  out of the "worked example" surface. No cross-probe imports.
- **Stateless.** No databases, no caches, no goroutines that outlive a
  request. Each handler is a pure function of the request.
- **Small.** A probe that grows past ~50 lines probably wants its own
  service, not a corner of this one.

## Configuration

Environment variables only — no CLI flags. Deployments inject env;
that's the contract kitchensink exists to exercise.

- `PORT` — listen port. Defaults to `8080`.

## Layout

```
svc/kitchensink/
├── main.go       — wires method+path → handler
├── hello/        — GET /hello, smoke test
├── env/          — GET /env, process environment
├── buildinfo/    — GET /buildinfo, value injected via -ldflags -X
├── principal/    — GET /principal, decodes X-Unkey-Principal
├── headers/      — GET /headers, echoes request headers
├── echo/         — POST /echo, echoes body verbatim
├── logs/         — POST /log, logs body at INFO
├── status/       — GET /status/{code}, returns arbitrary status
└── sleep/        — GET /sleep?d=<duration>, blocks before responding
```

## Adding a probe

1. Create `svc/kitchensink/<name>/handler.go`.
2. Write `func Handler(w http.ResponseWriter, r *http.Request)` with the
   behavior.
3. Register it in `main.go`'s `routes` map:

    ```go
    "GET /<name>": <name>.Handler,
    ```

Look at `hello/handler.go` for the minimum viable shape. Use
`http.ServeMux` pattern syntax in the registration key (Go 1.22+) —
`GET /foo`, `POST /bar/{id}`, etc.

## Running

Directly with Go:

```
go run ./svc/kitchensink
```

With a custom port:

```
PORT=9090 go run ./svc/kitchensink
```

Via Docker (build context is the repo root, not this directory):

```
docker build -f svc/kitchensink/Dockerfile -t kitchensink .
docker run --rm -p 8080:8080 kitchensink
```

Then:

```
curl localhost:8080/hello
curl localhost:8080/env
curl localhost:8080/status/503
```
