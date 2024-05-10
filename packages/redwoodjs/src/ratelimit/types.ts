import type { Logger } from "@redwoodjs/api/logger";
import type { MiddlewareRequest } from "@redwoodjs/vite/middleware";
import type { MiddlewareResponse } from "@redwoodjs/vite/middleware";
import type { RatelimitConfig } from "@unkey/ratelimit";

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
  ratelimit: withUnkeyRatelimitConfig;
  logger?: Logger;
};
