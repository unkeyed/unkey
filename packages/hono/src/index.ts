import { ErrorResponse, verifyKey } from "@unkey/api";
import type { Context, MiddlewareHandler } from "hono";
import { HTTPException } from "hono/http-exception";

export type UnkeyContext = {
  valid: boolean;
  ownerId?: string | undefined;
  meta?: unknown;
  expires?: number | undefined;
  remaining?: number | undefined;
  ratelimit?:
    | {
        limit: number;
        remaining: number;
        reset: number;
      }
    | undefined;
  code?:
    | "NOT_FOUND"
    | "RATE_LIMITED"
    | "FORBIDDEN"
    | "USAGE_EXCEEDED"
    | "UNAUTHORIZED"
    | "DISABLED"
    | undefined;
};

export type UnkeyConfig<TContext = Context> = {
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
  getKey?: (c: TContext) => string | undefined | Response;

  /**
   * Automatically return a custom response when a key is invalid
   */
  handleInvalidKey?: (c: TContext, result: UnkeyContext) => Response | Promise<Response>;

  /**
   * What to do if things go wrong
   */
  onError?: (c: TContext, err: ErrorResponse["error"]) => Response | Promise<Response>;
};

export function unkey(config?: UnkeyConfig): MiddlewareHandler {
  return async (c, next) => {
    const key = config?.getKey
      ? config.getKey(c)
      : c.req.header("Authorization")?.replace("Bearer ", "") ?? null;
    if (!key) {
      return c.json({ error: "unauthorized" }, { status: 401 });
    } else if (typeof key !== "string") {
      return key;
    }

    const res = await verifyKey(key);
    if (res.error) {
      if (config?.onError) {
        return config.onError(c, res.error);
      }
      throw new HTTPException(500, {
        message: `unkey error: [CODE: ${res.error.code}] - [REQUEST_ID: ${res.error.requestId}] - ${res.error.message} - read more at ${res.error.docs}`,
      });
    }

    if (!res.result.valid && config?.handleInvalidKey) {
      return config.handleInvalidKey(c, res.result);
    }

    c.set("unkey", res.result);
    await next();
  };
}
