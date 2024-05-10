import { Unkey } from "@unkey/api";
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
  if (!config) {
    throw new Error("withUnkey requires a configuration object");
  }

  if (!config.ratelimit && !config.auth) {
    throw new Error("withUnkey requires a ratelimit or auth configuration object");
  }

  const logger = config.logger || defaultLogger;

  if (config.auth) {
    return async (req: MiddlewareRequest, res: MiddlewareResponse) => {
      /**
       * Get key from request and return a response early if not found
       */
      const key = config.auth?.getKey
        ? config.auth.getKey(req)
        : req.headers.get("authorization")?.replace("Bearer ", "") ?? null;

      if (key === null) {
        res.body = "Unauthorized";
        res.status = 401;
        return res;
      }

      if (typeof key !== "string") {
        return key;
      }

      const unkey = new Unkey({
        rootKey: "public",
        wrapperSdkVersion: `@unkey/redwoodjs@${version}`,
        disableTelemetry: config.auth?.disableTelemetry,
      });

      const unkeyVerificationResult = await unkey.keys.verify(
        config.auth?.apiId ? { key, apiId: config.auth?.apiId } : { key },
      );

      if (unkeyVerificationResult.error) {
        if (config.auth?.onError) {
          return config.auth.onError(
            req,
            //unkeyVerificationResult.error
          );
        }
        console.error(
          `unkey error: [CODE: ${unkeyVerificationResult.error.code}] - [TRACE: ${unkeyVerificationResult.error.requestId}] - ${unkeyVerificationResult.error.message} - read more at ${unkeyVerificationResult.error.docs}`,
        );

        res.body = "Internal Server Error";
        res.status = 500;
      }

      if (!unkeyVerificationResult.result?.valid) {
        if (config.auth?.onInvalidKey) {
          return config.auth.onInvalidKey(
            req, //res.result
          );
        }

        res.body = "Unauthorized";
        res.status = 401;

        // return new NextResponse("Unauthorized", { status: 500 });
      }

      return res;
    };
  }

  if (config.ratelimit) {
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
  }

  return async (req: MiddlewareRequest, res: MiddlewareResponse) => {
    logger.error(req, "No configuration provided. Skipping middleware.");
    return res;
  };
};

export default withUnkey;
