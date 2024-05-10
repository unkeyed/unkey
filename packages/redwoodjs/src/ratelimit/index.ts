import type { Logger } from "@redwoodjs/api/logger";
import type { MiddlewareRequest } from "@redwoodjs/vite/middleware";
import type { MiddlewareResponse } from "@redwoodjs/vite/middleware";
import { Ratelimit } from "@unkey/ratelimit";
import type { withUnkeyOptions } from "./types";
import {
  defaultRatelimitErrorResponse,
  defaultRatelimitExceededResponse,
  defaultRatelimitIdentifier,
} from "./util";

const defaultLogger = require("abstract-logging") as Logger;

/**
 * withUnkey is RedwoodJS middleware that adds Unkey rate limiting to a route
 *
 * @see https://redwoodjs.com/docs/middleware#withUnkey
 * @see https://www.unkey.com/docs/apis/features/ratelimiting
 *
 * Provide the Unkey rate limit configuration and the path matcher to apply the rate limit to.
 *
 * @param options withUnkeyOptions: withUnkeyOptions;
 * @param options ratelimit: withUnkeyRatelimitConfig;
 * @param options ratelimit config: RatelimitConfig;
 *
 * You can provide optional custom functions to construct rate limit identifier,
 * rate limit exceeded response, and rate limit error response.
 *
 * @param options ratelimit getIdentifier?: (req: MiddlewareRequest) => string;
 * @param options ratelimit onExceeded?: (req: MiddlewareRequest) => MiddlewareResponse;
 * @param options ratelimit onError?: (req: MiddlewareRequest) => MiddlewareResponse;
 *
 * @param options logger?: Logger;
 *
 * @example
 * ```ts file="web/src/entry.server.tsx"
 *
 * import withUnkey from '@unkey/redwoodjs'
 * import type { withUnkeyOptions } from '@unkey/redwoodjs'
 *
 * export const registerMiddleware = () => {
 *  const options: withUnkeyOptions = {
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
 */
const withUnkey = (options: withUnkeyOptions) => {
  if (!options.ratelimit) {
    throw new Error("ratelimitConfig is required");
  }

  const unkey = new Ratelimit(options.ratelimit.config);

  return async (req: MiddlewareRequest, res: MiddlewareResponse) => {
    const logger = options.logger || defaultLogger;

    const getIdentifier = options.ratelimit?.getIdentifier || defaultRatelimitIdentifier;

    const onExceeded = options.ratelimit?.onExceeded || defaultRatelimitExceededResponse;

    const onError = options.ratelimit?.onError || defaultRatelimitErrorResponse;

    try {
      const identifier = getIdentifier(req);

      const ratelimit = await unkey.limit(identifier);

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
