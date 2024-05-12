import type { ErrorResponse, Unkey } from "@unkey/api";

import type { Logger } from "@redwoodjs/api/logger";
import type { MiddlewareRequest } from "@redwoodjs/vite/middleware";
import type { MiddlewareResponse } from "@redwoodjs/vite/middleware";

export type VerifyResponse = Awaited<ReturnType<InstanceType<typeof Unkey>["keys"]["verify"]>>;
export type UnkeyContext = VerifyResponse["result"];

export type ApiKeyMiddlewareConfig = {
  /**
   * The apiId to verify against.
   *
   * This will be required soon.
   */
  apiId?: string;

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
   * Return the key as string, or null if it doesn't exist.
   *
   * You can also override the response given to the caller by returning a `NextResponse`
   *
   * @default `req.headers.get("authorization")?.replace("Bearer ", "") ?? null`
   */
  getKey?: (req: MiddlewareRequest) => string;

  /**
   * Automatically return a custom response when a key is invalid
   */
  onInvalidKey?: (req: MiddlewareRequest, result: UnkeyContext) => MiddlewareResponse;

  /**
   * What to do if things go wrong
   */
  onError?: (req: MiddlewareRequest, err: ErrorResponse["error"]) => MiddlewareResponse;

  /*
   * RedwoodJS-compatible Logger to use for logging
   */
  logger?: Logger;
};
