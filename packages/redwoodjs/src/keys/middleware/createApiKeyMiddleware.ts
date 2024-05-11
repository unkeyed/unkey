import { Unkey } from "@unkey/api";
import { version } from "../../../package.json";

import { defaultLogger } from "../../index";

import type { Middleware, MiddlewareRequest, MiddlewareResponse } from "@redwoodjs/vite/middleware";
import type { ApiKeyMiddlewareConfig } from "./types";

/**
 * Get key tp verify from request using the custom getKey function
 * or the default authorization header
 */
export const getKey = (config: ApiKeyMiddlewareConfig, req: MiddlewareRequest): string | null => {
  return config.getKey
    ? config.getKey(req)
    : req.headers.get("authorization")?.replace("Bearer ", "") ?? null;
};

/**
 * Create RedwoodJS middleware to verify an API key using Unkey
 */
const createApiKeyMiddleware = (config: ApiKeyMiddlewareConfig): Middleware => {
  const logger = config.logger || defaultLogger;

  return async (req: MiddlewareRequest, res: MiddlewareResponse) => {
    // Get key from request and return
    const key = getKey(config, req);

    // return an unauthorized response if no key is found
    if (key === null) {
      res.body = "Unauthorized";
      res.status = 401;
      return res;
    }

    if (typeof key !== "string") {
      // why not error?
      logger.error(key, "Key is not a string");
      return key;
    }

    // Create an instance of Unkey
    const unkey = new Unkey({
      rootKey: "public",
      wrapperSdkVersion: `@unkey/redwoodjs@${version}`,
      disableTelemetry: config.disableTelemetry,
    });

    // if apiId is provided, verify against it otherwise just verify the key
    const unkeyVerificationResult = await unkey.keys.verify(
      config.apiId ? { key, apiId: config.apiId } : { key },
    );

    // if there is an error, log it and return an error response
    if (unkeyVerificationResult.error) {
      if (config.onError) {
        return config.onError(req, unkeyVerificationResult.error);
      }
      logger.error(
        `unkey error: [CODE: ${unkeyVerificationResult.error.code}] - [TRACE: ${unkeyVerificationResult.error.requestId}] - ${unkeyVerificationResult.error.message} - read more at ${unkeyVerificationResult.error.docs}`,
      );

      res.body = "Internal Server Error";
      res.status = 500;
    }

    // if the key is invalid, return an unauthorized response
    if (!unkeyVerificationResult.result?.valid) {
      if (config.onInvalidKey) {
        return config.onInvalidKey(req, unkeyVerificationResult.result);
      }

      res.body = "Unauthorized";
      res.status = 401;
    }

    // if the key is valid, continue
    return res;
  };
};

export default createApiKeyMiddleware;
