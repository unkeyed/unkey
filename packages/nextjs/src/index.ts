import { type ErrorResponse, verifyKey } from "@unkey/api";
import { type NextRequest, NextResponse } from "next/server";

export type WithUnkeyConfig = {
  /**
   * The apiId to verify against.
   *
   * This will be required soon.
   */
  apiId?: string;
  /**
   * How to get the key from the request
   * Usually the key is provided in an `Authorization` header, but you can do what you want.
   *
   * Return the key as string, or null if it doesn't exist.
   *
   * You can also override the response given to the caller by returning a `NextResponse`
   *
   * @default `req.headers.get("authorization")?.replace("Bearer ", "") ?? null`
   */
  getKey?: (req: NextRequest) => string | null | Response | NextResponse;

  /**
   * Automatically return a custom response when a key is invalid
   */
  handleInvalidKey?: (
    req: NextRequest,
    result: UnkeyContext,
  ) => Response | NextResponse | Promise<Response> | Promise<NextResponse>;

  /**
   * What to do if things go wrong
   */
  onError?: (
    req: NextRequest,
    err: ErrorResponse["error"],
  ) => Response | NextResponse | Promise<Response> | Promise<NextResponse>;
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
  code?:
    | "NOT_FOUND"
    | "RATE_LIMITED"
    | "FORBIDDEN"
    | "USAGE_EXCEEDED"
    | "UNAUTHORIZED"
    | "DISABLED"
    | undefined;
};

export type NextContext = { params: string };

export type NextRequestWithUnkeyContext = NextRequest & { unkey: UnkeyContext };

export function withUnkey(
  handler: (
    req: NextRequestWithUnkeyContext,
    context: NextContext,
  ) => Response | NextResponse | Promise<Response | NextResponse>,
  config?: WithUnkeyConfig,
) {
  return async (req: NextRequest, context: NextContext) => {
    /**
     * Get key from request and return a response early if not found
     */
    const key = config?.getKey
      ? config.getKey(req)
      : req.headers.get("authorization")?.replace("Bearer ", "") ?? null;
    if (key === null) {
      return NextResponse.json({ error: "unauthorized" }, { status: 401 });
    } else if (typeof key !== "string") {
      return key;
    }

    const res = await verifyKey(config?.apiId ? { key, apiId: config.apiId } : key);
    if (res.error) {
      if (config?.onError) {
        return config.onError(req, res.error);
      }
      console.error(
        `unkey error: [CODE: ${res.error.code}] - [TRACE: ${res.error.requestId}] - ${res.error.message} - read more at ${res.error.docs}`,
      );
      return new NextResponse("Internal Server Error", { status: 500 });
    }

    if (!res.result.valid) {
      if (config?.handleInvalidKey) {
        return config.handleInvalidKey(req, res.result);
      }

      console.log(`Invalid key - ${res.result.code}`);

      return new NextResponse("Unauthorized", { status: 500 });
    }

    // @ts-ignore
    req.unkey = res.result;

    return handler(req as NextRequestWithUnkeyContext, context);
  };
}
