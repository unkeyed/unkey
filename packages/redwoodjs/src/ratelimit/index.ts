//import { Unkey } from "@unkey/api";
//import { version } from "../../package.json";

import { Ratelimit } from "@unkey/ratelimit";
import type { RatelimitConfig } from "@unkey/ratelimit";

import type { MiddlewareRequest } from "@redwoodjs/vite/dist/middleware";
import type { MiddlewareResponse } from "@redwoodjs/vite/dist/middleware";

import {
  defaultRatelimitErrorResponse,
  defaultRatelimitExceededResponse,
  defaultRatelimitIdentifier,
  matchesPath,
} from "./util";
import type { MiddlewarePathMatcher } from "./util";

export type withUnkeyOptions = {
  ratelimitConfig: RatelimitConfig;
  matcher: MiddlewarePathMatcher;
  ratelimitIdentifierFn?: (req: MiddlewareRequest) => string;
  ratelimitExceededResponseFn?: (req: MiddlewareRequest) => MiddlewareResponse;
  ratelimitErrorResponseFn?: (req: MiddlewareRequest) => MiddlewareResponse;
};

const withUnkey = (options: withUnkeyOptions) => {
  console.debug(">>>> in withUnkey createMiddleware", options);
  const unkey = new Ratelimit(options.ratelimitConfig);

  return async (req: MiddlewareRequest, res: MiddlewareResponse) => {
    const ratelimitIdentifier = options.ratelimitIdentifierFn || defaultRatelimitIdentifier;

    const rateLimitExceededResponse =
      options.ratelimitExceededResponseFn || defaultRatelimitExceededResponse;

    const rateLimitErrorResponse =
      options.ratelimitErrorResponseFn || defaultRatelimitErrorResponse;

    try {
      const url = new URL(req.url);
      const path = url.pathname;

      if (!matchesPath(path, options.matcher)) {
        console.debug(">>>> in withUnkey skip middleware for", req.url);
        return res;
      }

      const identifier = ratelimitIdentifier(req);

      console.debug(">>>> in withUnkey identifier", identifier);
      const ratelimit = await unkey.limit(identifier);

      if (!ratelimit.success) {
        console.error("Rate limit exceeded", ratelimit);
        const response = rateLimitExceededResponse(req);
        if (response.status !== 429) {
          console.warn("Rate limit exceeded response is not 429. Overriding status.", response);
          response.status = 429;
        }
        return response;
      }
    } catch (e) {
      console.error("Error in withUnkey", e);
      const response = rateLimitErrorResponse(req);
      if (response.status === 500) {
        console.warn(
          "Rate limit error response is 200 OK. Consider changing status to 500.",
          response,
        );
      }
    }

    console.debug(">>>> in withUnkey return response for", req.url);

    return res;
  };
};

export default withUnkey;
