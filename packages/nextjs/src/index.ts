import { verifyKey } from "@unkey/api";
import { type NextFetchEvent, NextRequest, NextResponse } from "next/server";

export type WithUnkeyConfig = {
  /**
   * How to get the key from the request
   * Usually the key is provided in an `Authorization` header, but you can do what you want.
   *
   * Return the key as string, or null if it doesn't exist.
   *
   * You can also override the response given to the caller by returning a `NextResponse`
   *
   * @default `req.headers.get("Authorization")?.replace("Bearer ", "") ?? null`
   */
  getKey?: (req: NextRequest) => string | null | NextResponse;
};

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
  code?: "NOT_FOUND" | "RATELIMITED" | "FORBIDDEN" | "KEY_USAGE_EXCEEDED" | undefined;
};

export type NextRequestWithUnkeyContext = NextRequest & { unkey: UnkeyContext };

export function unstable__withUnkey(
  handler: (
    req: NextRequestWithUnkeyContext,
    nfe?: NextFetchEvent,
  ) => NextResponse | Promise<NextResponse>,
  config?: WithUnkeyConfig,
) {
  return async (req: NextRequest, nfe: NextFetchEvent) => {
    /**
     * Get key from request and return a response early if not found
     */
    const key = config?.getKey ? config.getKey(req) : null;
    if (key === null) {
      return NextResponse.json({ error: "unauthorized" }, { status: 401 });
    } else if (typeof key !== "string") {
      return key;
    }

    const verified = await verifyKey(key);
    if (verified.error) {
      console.error(
        "unkey error: [CODE: %s] - [TRACE: %s] - %s - read more at %s",
        verified.error.code,
        verified.error.requestId,
        verified.error.message,
        verified.error.docs,
      );
      return new NextResponse("Internal Server Error", { status: 500 });
    }

    const unkeyContext: UnkeyContext = {
      ...verified.result,
    };

    // @ts-ignore
    req.unkey = unkeyContext;

    return handler(req as NextRequestWithUnkeyContext, nfe);
  };
}
