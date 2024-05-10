import type { Logger } from "@redwoodjs/api/logger";
import type { MiddlewareRequest } from "@redwoodjs/vite/middleware";
import type { MiddlewareResponse } from "@redwoodjs/vite/middleware";
import type { RatelimitConfig } from "@unkey/ratelimit";

export type withUnkeyRatelimitConfig = {
  config: RatelimitConfig;
  getIdentifier?: (req: MiddlewareRequest) => string;
  onExceeded?: (req: MiddlewareRequest) => MiddlewareResponse;
  onError?: (req: MiddlewareRequest) => MiddlewareResponse;
};
export type withUnkeyOptions = {
  ratelimit?: withUnkeyRatelimitConfig;
  logger?: Logger;
};
