# deno server with unkey

A simple deno server with unkey.

## Run

```bash
deno run --allow-net --allow-env server.ts
```

## Test

```bash
curl http://localhost:8000 -H "Authorization: Bearer <KEY>"
```
