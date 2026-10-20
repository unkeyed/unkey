# Deploy demo API

A small Zen API for trying Unkey Deploy and API key management. It starts
without a database, root key, or other runtime service. It listens on `PORT`
(8080 by default) and returns JSON.

| Route | Result |
| --- | --- |
| `GET /` | Available endpoints and gateway setup instructions |
| `GET /health` | `{"status":"ok"}` |
| `GET /api/hello` | A greeting and request ID |

The API is public until you attach a gateway Key Auth policy. The app does not
verify keys itself, trust principal headers, or expose credentials or environment
variables. No keys or root tokens belong in the image.

## Run locally

From the repository root, use the repository's installed mise tools:

```bash
mise exec -- go run ./svc/deploydemo
curl http://localhost:8080/api/hello
mise exec -- rask ./svc/deploydemo
```

## Build the image

The build task compiles static binaries for `linux/amd64` and `linux/arm64`,
then packages them in the repository's non-root distroless base. It does not
push unless you pass `--push`.

```bash
mise run build-deploy-demo-image --load --tag unkey-deploy-demo:local
docker run --rm -p 8080:8080 unkey-deploy-demo:local
```

If your Docker image store cannot load multi-platform images, use `--push`
with a registry you control instead of `--load`.

## Protect the API with Unkey

1. Deploy the published image from **Projects > Create app > Container image**.
   Keep the runtime port at 8080. Use `/health` for HTTP health checks.
2. Create a keyspace and an API key in **API Management**, in the same workspace.
3. Open the deployed app's **Policies**, add a **Key Auth** policy, and select
   that keyspace. Match the path prefix `/api/`, use Bearer authentication, and
   set credit cost to 0. Leave `/` and `/health` public.
4. Redeploy so the policy takes effect.
5. Send a request without a key. The gateway must return 401, not 200.
6. Send a request with the API key. The gateway must allow the request, and the
   app must return 200.
7. Revoke the key in API Management and repeat the request. The gateway must
   reject it with 401. Add a permission requirement or per-key rate limit to
   try 403 and 429 responses.

```bash
export DEMO_URL='https://your-deployment-domain'
curl -i "$DEMO_URL/api/hello" -H 'Accept: application/json'
read -rs -p 'Demo API key: ' DEMO_API_KEY; echo
curl -i "$DEMO_URL/api/hello" \
  -H 'Accept: application/json' \
  -H "Authorization: Bearer $DEMO_API_KEY"
unset DEMO_API_KEY
```

Only use an end-user demo key, not a root key. The gateway rejects invalid
requests before they reach this container and records verification analytics.
See [gateway API key authentication](https://www.unkey.com/docs/compute/gateway/api-key-auth).

## Publish

After the workflow is merged into the default branch, the **Deploy Demo Image**
Depot workflow publishes
`ghcr.io/unkeyed/deploy-demo:vX.Y.Z` when a `deploy-demo/vX.Y.Z` tag is pushed.
It uses the same `GHCR_TOKEN` secret as the service release workflow.

After the first publication, an organization package admin must make the package
public in GitHub Packages. A public source repository does not make a GHCR image
public. Verify an anonymous pull before using the image as a dashboard example.
