# Verifying API Keys using Cloudflare Workers

This is a minimal worker that verifies API keys using Cloudflare Workers

## Install

```bash
pnpm install
```

## Init

If you haven't already, initialize the worker.

```bash
pnpm wrangler init
```

## Deploy

```bash
pnpm wrangler deploy
```

This will output the URL of the worker. Let's use it to send a request.


## Test

Go to [unkey.dev](https://unkey.dev/app) and create a new key. Then, send a request to the worker.


```bash
curl <URL_FROM_PREVIOUS_STEP> -H "Authorization: Bearer <KEY>"
```

It should return a 200 reponse with some data about the key.