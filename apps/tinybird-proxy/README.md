


<div align="center">
    <h1 align="center">tinybird-proxy</h1>
    <h5>/v0/events proxy to batch many requests before forwarding to Tinybird</h5>
</div>

<div align="center">
  <a href="https://unkey.com">unkey.com</a>
</div>
<br/>


This is an `/v0/events`-compatible proxy for the Tinybird API that batches many requests before forwarding them to Tinybird. This is useful for reducing the number of requests to Tinybird, since they have a 1k RPS ingest limit.

We are using cloudflare workers for our API and would regularly exceed the 1k RPS limit of tinybird, because we can not batch events inside the workers.
This is our solution.

Instead of sending single events to tinybird, we send them to the proxy, which will batch them and flush periodically to tinybird in a single request.


## Usage

### Using go

```bash
git clone git@github.com:unkeyed/unkey.git
cd unkey/apps/tinybird-proxy

// Set the environment variables
go run cmd/main.go
```

### Using docker

TODO: publish to ghcr

```bash
git clone git@github.com:unkeyed/unkey.git
cd unkey/apps/tinybird-proxy
docker build --platform=linux/amd64 -t tinybird-proxy .
docker run -p 8080:8080 -e TINYBIRD_TOKEN=<token> tinybird-proxy
```

## Config
All configration is done via environment variables.

```bash

# The port where the proxy will listen
# Optional, default is 8080
PORT=8080

# Tinybird credentials to ingest data
TINYBIRD_TOKEN=

# Optional, default is https://api.tinybird.co
TINYBIRD_BASE_URL=https://api.tinybird.co

# Up to how many rows to send to Tinybird in each request
# Optional, default is 100000
BATCH_SIZE=100000

# How many rows to buffer at most. If the buffer is full, the proxy will not accept more data until it's flushed
# Optional, default is 1000000
BUFFER_SIZE=1000000

# How often to flush the buffer in milliseconds, even if the batch size is not reached
# Optional, default is 1000
FLUSH_INTERVAL=1000

# Authenticate incoming requests with a token
# Example: AUTHORIZATION_TOKEN=secret means that the Authorization header must be set to "Bearer secret"
# Optional
AUTHORIZATION_TOKEN=

# A unique id for the node where the proxy is running
# Optional, used in metrics
NODE_ID=

# The proxy can emit metrics about the ingestion process
# The metrics are sent to a Tinybird datasource.
# Optional
TINYBIRD_METRICS_DATASOURCE=
  ```



## Optional: Create metrics datasource

```sql
VERSION 1
SCHEMA >
    `nodeId` String `json:$.nodeId`,
    `requests` Int64 `json:$.requests`,
    `rows` Int64 `json:$.rows`,
    `flushes` Int64 `json:$.flushes`,
    `time` Int64 `json:$.time`

ENGINE "MergeTree"
ENGINE_SORTING_KEY "nodeId, time"
```

And provide the datasource name as environment variable:
```bash
TINYBIRD_METRICS_DATASOURCE=<datasource name>
```

## Send data

Send data as you would to the Tinybird API, but to the proxy instead.
Use the url and the authorization token of the proxy.

```bash
curl \
-H "Authorization: Bearer <AUTHORIZATION_TOKEN>" \
-d '{"date": "2020-04-05 00:05:38", "city": "Chicago"}' \
'<TINYBIRD_PROXY_URL>/v0/events?name=events_test'
```

### Using @chronark/zod-bird


```ts
import { Tinybird } from '@chronark/zod-bird'

const tb = new Tinybird({ token: "AUTHORIZATION_TOKEN", baseUrl: "TINYBIRD_PROXY_URL" })
```

Keep in mind this proxy is only handling the `/v0/events` endpoint. Use a different instance for querying data.
