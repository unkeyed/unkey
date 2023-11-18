<div align="center">
    <h1 align="center">@unkey/hono</h1>
    <h5>Hono.js middleware for authenticating API keys</h5>
</div>

<div align="center">
  Inspired by <a href="https://www.openstatus.dev/blog/secure-api-with-unkey">openstatus.dev/blog/secure-api-with-unkey</a>
</div>
<br/>

Check out the docs at [unkey.dev/docs](https://unkey.dev/docs/libraries/ts/hono).

Here's just an example:

```ts
import { serve } from "@hono/node-server";
import { Hono, MiddlewareHandler } from "hono";
import { UnkeyContext, unkey } from "@unkey/hono";

type Variables = {
	customFn: (str: string) => string;
	customData: Record<string, unknown>;
	unkey: UnkeyContext;
};


const app = new Hono<{ Variables: Variables }>();


app.use(
	"*",
	unkey({
		getKey: (c) => {
			// Parse Api key from client
			let header = c.req.header("Authorization");
			let token = header?.split(" ")[1];
			if (!token) {
				// Customize the error message or just return undefined
				return c.json({
					status: false,
					message: "Forbidden, You are unauthorized, provide a valid api key",
					data: [],
				});
			}

			return token;
		},
		handleInvalidKey(c, result) {
			// Handle Invalid key
			if (!result.valid) {
				return c.json(
					{
						status: false,
						message: "Forbidden, You are unauthorized, probably an invalid key",
						data: [],
					},
					403
				);
			}

			if (result.code === "RATELIMITED") {
				return c.json(
					{
						status: false,
						message: "Too many requests, please try again later",
						data: [],
					},
					429
				);
			}

			return c.json(
				{
					status: false,
					message: "Internal Server error, please check back later",
					data: [],
				},
				500
			);
		},
		onError: (c, err) => {
			// Handle error type here
			if (err.code === "INTERNAL_SERVER_ERROR") {
				return c.json(
					{
						status: false,
						message: "Internal Server error, third party error",
						data: [],
					},
					500
				);
			}
			return c.json(
				{
					status: false,
					message: "Internal Server error, please check back later",
					data: [],
				},
				500
			);
		},
	})
);

app.get("/user", async (c) => {
  let unKeyMetaData = c.get("unkey");
	console.log("Unkey metadata", unKeyMetaData);

	return c.text("Protected route!");
});
```
