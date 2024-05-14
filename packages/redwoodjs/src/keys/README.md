# RedwoodJS Api Key Verification

`createApiKeyMiddleware` is middleware function for RedwoodJS used to validate API keys sent in the request headers with [Unkey](https://www.unkey.com/docs).

By default, Unkey verifies a key provided in the request's authorization header: `"Authorization: Bearer unkey_xxx"`. How Unkey extracts the key to verify can be customized using the `getKey` function option.

In the future, support for key verification in GraphQL operations and other RedwoodJS functions will be added.

## Usage

Here's a basic example of how to use `createApiKeyMiddleware`:

```ts file="web/src/entry.server.tsx"
import createApiKeyMiddleware from "@unkey/redwoodjs";
import type { ApiKeyMiddlewareConfig } from "@unkey/redwoodjs";

export const registerMiddleware = () => {
  const config: ApiKeyMiddlewareConfig = {
    apiId: "my-app-id",
  };

  const middleware = createApiKeyMiddleware(config);

  return [middleware];
};
```

In this example, `createApiKeyMiddleware` is used as a global middleware. It will validate the API key for all incoming requests.

## Configuration

`createApiKeyMiddleware` can be configured by providing `ApiKeyMiddlewareConfig` configuration options when creating the middleware.

### Custom getKey

In this example, `createApiKeyMiddleware` will look for the API key in the 'X-API-KEY' header and validate it via a custom `getKey` function.

```ts file="web/src/entry.server.tsx"
import createApiKeyMiddleware from "@unkey/redwoodjs";
import type { ApiKeyMiddlewareConfig } from "@unkey/redwoodjs";

export const registerMiddleware = () => {
  const config: ApiKeyMiddlewareConfig = {
    apiId: 'my-app-id'
    getKey: (req) => {
    return req.headers.get("x-api-key") ?? "";
    },
  };

  const middleware = createApiKeyMiddleware(config);

  return [middleware];
};
```

### Custom onInvalidKey

In this example, `createApiKeyMiddleware` respond with a custom message and status if the key is invalid.

```ts file="web/src/entry.server.tsx"
import createApiKeyMiddleware from "@unkey/redwoodjs";
import type { ApiKeyMiddlewareConfig } from "@unkey/redwoodjs";

export const registerMiddleware = () => {
  const config: ApiKeyMiddlewareConfig = {
    apiId: 'my-app-id'
    onInvalidKey: (_req, _result) => {
        return new MiddlewareResponse("Custom forbidden", { status: 403 });
    },
  };

  const middleware = createApiKeyMiddleware(config);

  return [middleware];
};
```

### Error Handling

If the API key is missing or invalid, `createApiKeyMiddleware` will send a 401 Unauthorized response and stop the request from being processed further.

You can customize this behavior by providing your own error handling function in the options object:

```ts file="web/src/entry.server.tsx"
import createApiKeyMiddleware from "@unkey/redwoodjs";
import type { ApiKeyMiddlewareConfig } from "@unkey/redwoodjs";

export const registerMiddleware = () => {
  const config: ApiKeyMiddlewareConfig = {
    apiId: "my-app-id",
    onError: (_req, _err) =>
      new MiddlewareResponse("Custom unavailable", { status: 503 }),
  };

  const middleware = createApiKeyMiddleware(config);

  return [middleware];
};
```
