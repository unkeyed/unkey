
<div align="center">
    <h1 align="center">@unkey/hono</h1>
    <h5>Hono.js middleware for authenticating API keys</h5>
</div>

<div align="center">
  <a href="https://unkey.dev">unkey.dev</a>
</div>
<br/>



Check out the docs at [docs.unkey.dev](https://docs.unkey.dev)

/**
   * The name in the context where the verification response is written to.
   *
   * You can later retrieve it with `c.get(..)`
   * Don't forget to add this to your `Variables type` like so
   * ```ts
   * import type { UnkeyContext } from "@unkey/hono"
   * type Variables = {
   *   unkey: UnkeyContext
   * }
   *
   * new Hono<{ Variables: Variables }>()
   * ```
   *
   * For more information how to read from the context, see: https://hono.dev/api/context#set-get
   */
  TODO?: "add this to docs instead";