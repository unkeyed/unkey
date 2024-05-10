import type { Logger } from "@redwoodjs/api/logger";
import type { MiddlewareRequest } from "@redwoodjs/vite/middleware";
import type { MiddlewareResponse } from "@redwoodjs/vite/middleware";
import type { RatelimitConfig } from "@unkey/ratelimit";

export type withUnkeyApiKeyConfig = {
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
  onInvalidKey?: (req: MiddlewareRequest) => MiddlewareResponse;

  /**
   * What to do if things go wrong
   */
  onError?: (req: MiddlewareRequest) => MiddlewareResponse;
};

export type withUnkeyRatelimitConfig = {
  /**
   * The Unkey configuration for the rate limiter
   */
  config: RatelimitConfig;

  /**
   * Custom function to get the identifier for the rate limiter
   */
  getIdentifier?: (req: MiddlewareRequest) => string;

  /**
   * Custom function to handle when the rate limit is exceeded
   */
  onExceeded?: (req: MiddlewareRequest) => MiddlewareResponse;

  /**
   * Custom function to handle when an error occurs
   */
  onError?: (req: MiddlewareRequest) => MiddlewareResponse;
};

export type withUnkeyConfig = {
  auth?: withUnkeyApiKeyConfig;
  ratelimit?: withUnkeyRatelimitConfig;
  logger?: Logger;
};
