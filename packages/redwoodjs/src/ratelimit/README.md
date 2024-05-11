# RedwoodJS Rate Limiting

RedwoodJS supports Unkey rate limiting via:

- middleware with Server-Side Rendering (SSR) enabled

In the future, support for rate limiting GraphQL operations and other RedwoodJS functions will be added.

# Middleware

Implement request rate limiting in your RedwoodJS application using the `createRatelimitMiddleware` from Unkey.

<Note>
  This middleware is only available when Server-Side Rendering (SSR) is enabled
  in RedwoodJS.
</Note>

To learn more about this middleware, visit the [RedwoodJS Middleware documentation](https://www.unkey.com/docs/libraries/ts/redwoodjs/middleware/ratelimiting).

## Setup

To get started with standalone rate limiting using Unkey, see the [Rate Limiting Onboarding Guide](https://www.unkey.com/docs/onboarding/onboarding-ratelimiting).

**Important**: Ensure that your `UNKEY_ROOT_KEY` is set in your `.env` file for rate limiting.

## Examples

The `createRatelimitMiddleware` function allows for extensive customization to suit various use cases. Below are examples demonstrating how to:

- Implement pattern matches to enforce rate limits on specific paths.
- Utilize a custom identifier generator function.
- Customize responses when rate limits are exceeded.
- Provide custom error responses.

### Rate Limit All Requests

To apply a rate limit to all requests, register the middleware globally without route matching:

```ts file="web/src/entry.server.tsx"
import createRatelimitMiddleware from "@unkey/redwoodjs";
import type { RatelimitMiddlewareConfig } from "@unkey/redwoodjs";

export const registerMiddleware = () => {
  const config: RatelimitMiddlewareConfig = {
    rootKey: process.env.UNKEY_ROOT_KEY,
    namespace: "my-app",
    limit: 1,
    duration: "30s",
    async: true,
  };

  const middleware = createRatelimitMiddleware(config);

  return [middleware];
};
```

### Basic Route Matching

To rate limit requests on specific routes, register the middleware with a pattern match:

```ts file="web/src/entry.server.tsx"
import createRatelimitMiddleware from "@unkey/redwoodjs";
import type { RatelimitMiddlewareConfig } from "@unkey/redwoodjs";

export const registerMiddleware = () => {
  const config: RatelimitMiddlewareConfig = {
    config: {
      rootKey: process.env.UNKEY_ROOT_KEY,
      namespace: "my-app",
      limit: 1,
      duration: "30s",
      async: true,
    },
  };

  const middleware = createRatelimitMiddleware(config);

  return [[middleware, "/blog-post/:slug(\\d{1,})"]];
};
```

To handle multiple patterns, either compose a complex expression

```ts
return [[middleware, "/rss.(xml|atom|json)"]];
```

or register multiple patterns:

```ts
return [
  [middleware, "/blog-post/:slug(\\d{1,})"],
  [middleware, "/admin"],
];
```

### With Custom Identifier Function and Third Party Authentication

Customize the identifier function to utilize user authentication status, such as with Supabase.

Here, we use a custom identifier function `supabaseRatelimitIdentifier` that:

- checks is the request is authenticated
- constructs the identifier `sub` from the current user, since here the currentUser will be a JWT where the user id is the `sub` claim
- registers `supabaseAuthMiddleware` before `middleware` so the request can be authenticated before determining limits

```ts file="web/src/entry.server.ts"
import createSupabaseAuthMiddleware from "@redwoodjs/auth-supabase-middleware";
import createRatelimitMiddleware from "@unkey/redwoodjs";
import type { RatelimitMiddlewareConfig } from "@unkey/redwoodjs";
import type { MiddlewareRequest } from "@redwoodjs/vite/middleware";
import type { TagDescriptor } from "@redwoodjs/web";

import App from "./App";
import { Document } from "./Document";

// eslint-disable-next-line no-restricted-imports
import { getCurrentUser } from "$api/src/lib/auth";

interface Props {
  css: string[];
  meta?: TagDescriptor[];
}

export const supabaseRatelimitIdentifier = (req: MiddlewareRequest) => {
  const authContext = req?.serverAuthContext?.get();
  console.log(">>>> in supabaseRatelimitIdentifier", authContext);
  const identifier = authContext?.isAuthenticated
    ? (authContext.currentUser?.sub as string) || "anonymous-user"
    : "192.168.1.1";
  return identifier;
};

export const registerMiddleware = () => {
  const config: RatelimitMiddlewareConfig = {
    config: {
      rootKey: process.env.UNKEY_ROOT_KEY,
      namespace: "my-app",
      limit: 1,
      duration: "30s",
      async: true,
    },
    getIdentifier: supabaseRatelimitIdentifier,
  };
  const middleware = createRatelimitMiddleware(config);
  const supabaseAuthMiddleware = createSupabaseAuthMiddleware({
    getCurrentUser,
  });

  return [supabaseAuthMiddleware, [middleware, "/blog-post/:slug(\\d{1,})"]];
};
```

### Custom Rate Limit Exceeded and Error Responses

Define custom responses for exceeded limits and errors:

```ts
export const registerMiddleware = () => {
  const config: RatelimitMiddlewareConfig = {
    config: {
      rootKey: process.env.UNKEY_ROOT_KEY,
      namespace: "my-app",
      limit: 1,
      duration: "30s",
      async: true,
    },
    onExceeded: (_req: MiddlewareRequest) => {
      return new MiddlewareResponse("Custom Rate limit exceeded message", {
        status: 429,
      });
    },
    onError: (_req: MiddlewareRequest) => {
      return new MiddlewareResponse(
        "Custom Error message when rate limiting fails",
        {
          status: 500,
        }
      );
    },
  };
  const middleware = createRatelimitMiddleware(config);

  return [middleware];
};
```
