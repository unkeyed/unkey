<div align="center">
    <h1 align="center">@unkey/api</h1>
    <h5>`@unkey/api` is a TypeScript client for Unkey. If you prefer a typed experience over calling HTTP endpoints directly, this SDK is for you.</h5>
</div>

<div align="center">
  <a href="https://www.unkey.com/docs/libraries/ts/sdk/overview">Documentation</a>
</div>
<br/>

## Installation

```bash
npm install @unkey/api
```

## Quickstart

1.  Create a new Unkey Root Key in the settings.
2.  Use the root key to initialize the client:

```ts
import { Unkey } from "@unkey/api";

const unkey = new Unkey({ rootKey: "<UNKEY_ROOT_KEY>" });
```

**Important:** Always keep your root key safe and reset it if you suspect it has been compromised.

## Usage

### Verifying a Key

```ts
import { verifyKey } from "@unkey/api";

const { result, error } = await verifyKey("key_123");

if (error) {
  console.error(error.message);
  // Handle potential network or bad request error
  // A link to our docs will be in the `error.docs` field
  return;
}

if (!result.valid) {
  // Do not grant access
  return;
}

// Process request
console.log(result);
```

## Response Format

All methods return either an `error` or a `result` field, never both and never none. This approach helps with proper error handling.

### Success Response

```ts
{
  result: T; // The result depends on what method you called
}
```

### Error Response

```ts
{
  error: {
    message: string;
    docs: string; // URL to relevant documentation
  }
}
```

## Configuration Options

### Base URL

You can customize the base URL for all requests:

```ts
const unkey = new Unkey({
  rootKey: "<UNKEY_ROOT_KEY>",
  baseUrl: "https://my.domain",
});
```

### Retries

Configure retry behavior for network errors:

```ts
const unkey = new Unkey({
  rootKey: "<UNKEY_ROOT_KEY>",
  retry: {
    attempts: 3,
    backoff: (retryCount) => retryCount * 1000,
  },
});
```

### Disable Telemetry

To opt out of anonymous telemetry data collection:

```ts
const unkey = new Unkey({
  rootKey: "<UNKEY_ROOT_KEY>",
  disableTelemetry: true,
});
```

### Documentation

[Read the full documentation](https://www.unkey.com/docs/libraries/ts/sdk/overview)
