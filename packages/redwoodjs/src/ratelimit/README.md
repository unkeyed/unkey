# withUnkey

RedwoodJS Middleware to rate limit requests

## Setup

See [Rate limiting Onboarding](https://www.unkey.com/docs/onboarding/onboarding-ratelimiting) to get started with standalone [rate limiting](https://www.unkey.com/docs/apis/features/ratelimiting) from [Unkey](https://www.unkey.com).

Note: Be sure to set your `UNKEY_ROOT_KEY` or key to be used for rate limiting in an `.env` file.

## Examples

### With Third Party Authentication like Supabase

Here, we use a custom identifier function `supabaseRatelimitIdentifier` that:

- checks is the request is authenticated
- constructs the identifier `sub` from the current user, since here the currentUser will be a JWT where the user id is the `sub` claim
- registers `supabaseAuthMiddleware` before `unkeyMiddleware` so the request can be authenticated before determining limits

```file="web/entry.server.ts"
import createSupabaseAuthMiddleware from '@redwoodjs/auth-supabase-middleware'
import type { MiddlewareRequest } from '@redwoodjs/vite/middleware'
import type { TagDescriptor } from '@redwoodjs/web'

import App from './App'
import { Document } from './Document'
import withUnkey from '@unkey/redwoodjs'
import type { withUnkeyOptions } from '@unkey/redwoodjs'

// eslint-disable-next-line no-restricted-imports
import { getCurrentUser } from '$api/src/lib/auth'

interface Props {
  css: string[]
  meta?: TagDescriptor[]
}

export const supabaseRatelimitIdentifier = (req: MiddlewareRequest) => {
  const authContext = req?.serverAuthContext?.get()
  console.log('>>>> in supabaseRatelimitIdentifier', authContext)
  const identifier = authContext?.isAuthenticated
    ? (authContext.currentUser?.sub as string) || 'anonymous-user'
    : '192.168.1.1'
  return identifier
}

export const registerMiddleware = () => {
  const options: withUnkeyOptions = {
    ratelimitConfig: {
      rootKey: process.env.UNKEY_ROOT_KEY,
      namespace: 'my-app',
      limit: 1,
      duration: '30s',
      async: true,
    },
    matcher: ['/blog-post/:slug(\\d{1,})'],
    ratelimitIdentifierFn: supabaseRatelimitIdentifier,
  }
  const unkeyMiddleware = withUnkey(options)
  const supabaseAuthMiddleware = createSupabaseAuthMiddleware({
    getCurrentUser,
  })
  return [supabaseAuthMiddleware, unkeyMiddleware]
}

interface Props {
  css: string[]
  meta?: TagDescriptor[]
}

export const ServerEntry: React.FC<Props> = ({ css, meta }) => {
  return (
    <Document css={css} meta={meta}>
      <App />
    </Document>
  )
}
```

## Custom Rate Limit Exceeded Response

TODO

## Custom Rate Limit Error Response

TODO
