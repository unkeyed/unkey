//import { Unkey } from "@unkey/api";
//import { version } from "../../package.json";

import type { Logger } from "@redwoodjs/api/logger";
import type { MiddlewareRequest } from "@redwoodjs/vite/middleware";
import type { MiddlewareResponse } from "@redwoodjs/vite/middleware";
import { Ratelimit } from "@unkey/ratelimit";
import type { RatelimitConfig } from "@unkey/ratelimit";
import {
  defaultRatelimitErrorResponse,
  defaultRatelimitExceededResponse,
  defaultRatelimitIdentifier,
  matchesPath,
} from "./util";
import type { MiddlewarePathMatcher } from "./util";

const defaultLogger = require("abstract-logging") as Logger;

export type withUnkeyOptions = {
  ratelimitConfig: RatelimitConfig;
  matcher: MiddlewarePathMatcher;
  logger?: Logger;
  ratelimitIdentifierFn?: (req: MiddlewareRequest) => string;
  ratelimitExceededResponseFn?: (req: MiddlewareRequest) => MiddlewareResponse;
  ratelimitErrorResponseFn?: (req: MiddlewareRequest) => MiddlewareResponse;
};

const withUnkey = (options: withUnkeyOptions) => {
  const unkey = new Ratelimit(options.ratelimitConfig);

  return async (req: MiddlewareRequest, res: MiddlewareResponse) => {
    const logger = options.logger || defaultLogger;
    logger.fatal(">>> Unkey bopom");
    const ratelimitIdentifier = options.ratelimitIdentifierFn || defaultRatelimitIdentifier;

    const rateLimitExceededResponse =
      options.ratelimitExceededResponseFn || defaultRatelimitExceededResponse;

    const rateLimitErrorResponse =
      options.ratelimitErrorResponseFn || defaultRatelimitErrorResponse;

    try {
      const url = new URL(req.url);
      const path = url.pathname;

      if (!matchesPath(path, options.matcher)) {
        return res;
      }

      const identifier = ratelimitIdentifier(req);

      const ratelimit = await unkey.limit(identifier);

      if (!ratelimit.success) {
        logger.debug("Rate limit exceeded", {
          ratelimit,
          url,
          path,
          identifier,
        });
        const response = rateLimitExceededResponse(req);
        if (response.status !== 429) {
          logger.warn("Rate limit exceeded response is not 429. Overriding status.", response);
          response.status = 429;
        }
        return response;
      }
    } catch (e) {
      logger.error("Error in withUnkey", e);
      const errorResponse = rateLimitErrorResponse(req);
      if (errorResponse.status !== 500) {
        logger.warn(
          `Rate limit error response is ${errorResponse.status}. Consider changing status to 500.`,
          errorResponse,
        );
      }

      return errorResponse;
    }

    return res;
  };
};

export default withUnkey;
