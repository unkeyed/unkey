import { Ratelimit } from "@unkey/ratelimit";

import { version } from "../../package.json";

import {
  defaultRatelimitErrorResponse,
  defaultRatelimitExceededResponse,
  defaultRatelimitIdentifier,
} from "./util";

import type { Logger } from "@redwoodjs/api/logger";

import type { MiddlewareRequest, MiddlewareResponse } from "@redwoodjs/vite/middleware";
import type { withUnkeyConfig } from "./types";

const defaultLogger = require("abstract-logging") as Logger;

/**
 * withUnkey is RedwoodJS middleware that adds Unkey rate limiting to a route
 *
 * @see https://redwoodjs.com/docs/middleware#withUnkey
 * @see https://www.unkey.com/docs/apis/features/ratelimiting
 *
 * Provide the Unkey rate limit configuration and the path matcher to apply the rate limit to.
 *
 * @param config withUnkeyOptions: withUnkeyOptions;
 * @param config ratelimit: withUnkeyRatelimitConfig;
 * @param config ratelimit config: RatelimitConfig;
 *
 * You can provide optional custom functions to construct rate limit identifier,
 * rate limit exceeded response, and rate limit error response.
 *
 * @param config ratelimit getIdentifier?: (req: MiddlewareRequest) => string;
 * @param config ratelimit onExceeded?: (req: MiddlewareRequest) => MiddlewareResponse;
 * @param config ratelimit onError?: (req: MiddlewareRequest) => MiddlewareResponse;
 *
 * @param config logger?: Logger;
 *
 * @example
 * ```ts file="web/src/entry.server.tsx"
 *
 * import withUnkey from '@unkey/redwoodjs'
 * import type { withUnkeyConfig } from '@unkey/redwoodjs'
 *
 * export const registerMiddleware = () => {
 *  const config: withUnkeyConfig = {
 *     ratelimit: {
 *       config: {
 *         rootKey: process.env.UNKEY_ROOT_KEY,
 *         namespace: 'my-app',
 *         limit: 1,
 *         duration: '30s',
 *         async: true,
 *       },
 *     }
 *   }
 *
 *   const unkeyMiddleware = withUnkey(options)
 *
 *   return [unkeyMiddleware]
 * }
 * ```
 *
 */
const withUnkey = (config: withUnkeyConfig) => {
  const logger = config.logger || defaultLogger;

  const unkeyRateLimiter = new Ratelimit(config.ratelimit.config);

  return async (req: MiddlewareRequest, res: MiddlewareResponse) => {
    const getIdentifier = config.ratelimit?.getIdentifier || defaultRatelimitIdentifier;

    const onExceeded = config.ratelimit?.onExceeded || defaultRatelimitExceededResponse;

    const onError = config.ratelimit?.onError || defaultRatelimitErrorResponse;

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
      logger.error("Error in withUnkey", e);
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

export default withUnkey;
