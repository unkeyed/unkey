<div align="center">
    <h1 align="center">@unkey/ratelimit</h1>
    <h5>@unkey/ratelimit is a library for fast global ratelimiting in serverless functions.</h5>
</div>

<div align="center">
  <a href="https://www.unkey.com/docs/libraries/ts/ratelimit">Documentation</a>
</div>
<br/>

## Installation

```bash
npm install @unkey/ratelimit
```

## Quickstart

1.  Configure your ratelimiter:

```ts
import { Ratelimit } from "@unkey/ratelimit";

const unkey = new Ratelimit({
  rootKey: process.env.UNKEY_ROOT_KEY,
  namespace: "my-app",
  limit: 10,
  duration: "30s",
  async: true,
});
```

2.  Use it:

```ts
async function handler(request) {
  const identifier = request.getUserId(); // or IP or anything else you want

  const ratelimit = await unkey.limit(identifier);
  if (!ratelimit.success) {
    return new Response("try again later", { status: 429 });
  }

  // handle the request here
}
```

## Making it Bullet Proof

To ensure reliability, you can configure timeout and error handling:

```ts
import { Ratelimit } from "@unkey/ratelimit";

const fallback = (identifier: string) => ({
  success: true,
  limit: 0,
  reset: 0,
  remaining: 0,
});

const unkey = new Ratelimit({
  // ... standard configuration
  timeout: {
    ms: 3000, // only wait 3s at most before returning the fallback
    fallback,
  },
  onError: (err, identifier) => {
    console.error(`${identifier} - ${err.message}`);
    return fallback(identifier);
  },
});
```

## API Overview

Create a new instance for ratelimiting by providing the necessary configuration.

```ts
new Ratelimit(config: RatelimitConfig)
```

Check whether a specific identifier is currently allowed to do something or if they have currently exceeded their limit.

```ts
.limit(identifier: string, opts: LimitOptions): Promise<RatelimitResponse>
```

### Documentation

[Read the full documentation](https://www.unkey.com/docs/libraries/ts/ratelimit)
