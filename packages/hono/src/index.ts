import { type ErrorResponse, Unkey } from "@unkey/api";
import type { Context, MiddlewareHandler } from "hono";
import { HTTPException } from "hono/http-exception";
import { version } from "../package.json";

type VerifyResponse = Awaited<ReturnType<InstanceType<typeof Unkey>["keys"]["verify"]>>;
export type UnkeyContext = VerifyResponse["result"];

export type UnkeyConfig = {
  /**
   * The apiId to verify against. Only keys belonging to this api will be valid.
   */
  apiId: string;

  /**
   *
   * By default telemetry data is enabled, and sends:
   * runtime (Node.js / Edge)
   * platform (Node.js / Vercel / AWS)
   * SDK version
   */
  disableTelemetry?: boolean;

  /**
   * How to get the key from the request
   * Usually the key is provided in an `Authorization` header, but you can do what you want.
   *
   * Return the key as string, or undefined if it doesn't exist.
   *
   * You can also override the response given to the caller by returning a `Response`
   *
   * @default `c.req.header("Authorization")?.replace("Bearer ", "")`
   */
  getKey?: (c: Context) => string | undefined | Response;

  /**
   * Automatically return a custom response when a key is invalid
   */
  handleInvalidKey?: (c: Context, result: UnkeyContext) => Response | Promise<Response>;

  /**
   * What to do if things go wrong
   */
  onError?: (c: Context, err: ErrorResponse["error"]) => Response | Promise<Response>;
};

export function unkey(config: UnkeyConfig): MiddlewareHandler {
  return async (c, next) => {
    const key = config.getKey
      ? config.getKey(c)
      : (c.req.header("Authorization")?.replace("Bearer ", "") ?? null);
    if (!key) {
      return c.json({ error: "unauthorized" }, { status: 401 });
    }
    if (typeof key !== "string") {
      return key;
    }

    const unkeyInstance = new Unkey({
      rootKey: "public",
      disableTelemetry: config.disableTelemetry,
      wrapperSdkVersion: `@unkey/hono@${version}`,
    });

    const res = await unkeyInstance.keys.verify({ key, apiId: config.apiId });
    if (res.error) {
      if (config.onError) {
        return config.onError(c, res.error);
      }
      throw new HTTPException(500, {
        message: `unkey error: [CODE: ${res.error.code}] - [REQUEST_ID: ${res.error.requestId}] - ${res.error.message} - read more at ${res.error.docs}`,
      });
    }

    if (!res.result.valid && config.handleInvalidKey) {
      return config.handleInvalidKey(c, res.result);
    }

    c.set("unkey", res.result);
    await next();
  };
}
