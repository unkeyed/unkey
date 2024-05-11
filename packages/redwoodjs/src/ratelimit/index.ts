import { Ratelimit } from "@unkey/ratelimit";

import { version } from "../../package.json";

import {
  defaultRatelimitErrorResponse,
  defaultRatelimitExceededResponse,
  defaultRatelimitIdentifier,
} from "./util";

import { defaultLogger } from "../index";

import type { Middleware, MiddlewareRequest, MiddlewareResponse } from "@redwoodjs/vite/middleware";
import type { RatelimitMiddlewareConfig } from "./types";

/**
 * createRatelimitMiddleware creates RedwoodJS middleware
 * that rate limits requests using Unkey's rate limiter.
 *
 * You can provide optional custom functions to construct rate limit identifier,
 * rate limit exceeded response, and rate limit error response.
 *
 * @see https://www.unkey.com/docs/apis/features/ratelimiting
 *
 */
const createRatelimitMiddleware = ({
  config,
  getIdentifier = defaultRatelimitIdentifier,
  onExceeded = defaultRatelimitExceededResponse,
  onError = defaultRatelimitErrorResponse,
  logger = defaultLogger,
}: RatelimitMiddlewareConfig): Middleware => {
  const unkeyRateLimiter = new Ratelimit(config);

  return async (req: MiddlewareRequest, res: MiddlewareResponse) => {
    try {
      const identifier = getIdentifier(req);

      const ratelimit = await unkeyRateLimiter.limit(identifier);

      if (!ratelimit.success) {
        logger.debug("Rate limit exceeded", {
          ratelimit,
          identifier,
        });
        const response = onExceeded(req);
        if (response.status !== 429) {
          logger.warn("Rate limit exceeded response is not 429. Overriding status.", response);
          response.status = 429;
        }
        return response;
      }
    } catch (e) {
      logger.error("Error in unkeyRateLimitMiddleware", e);
      const errorResponse = onError(req);
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

export default createRatelimitMiddleware;
