
<div align="center">
    <h1 align="center">@unkey/hono</h1>
    <h5>Hono.js middleware for authenticating API keys</h5>
</div>

<div align="center">
  Inspired by <a href="https://www.openstatus.dev/blog/secure-api-with-unkey">openstatus.dev/blog/secure-api-with-unkey</a>
</div>
<br/>



Check out the docs at [unkey.dev/docs](https://unkey.com/docs/libraries/ts/hono).


Here's just an example:

```ts
import { Hono } from "hono"
import { UnkeyContext, unkey } from "@unkey/hono";

const app = new Hono<{ Variables: { unkey: UnkeyContext } }>();

app.use("*", unkey());


app.get("/somewhere", (c) => {
  // access the unkey response here to get metadata of the key etc
  const ... = c.get("unkey")

  return c.text("yo")
})
``
