import { Unkey } from "@unkey/api";
import { version } from "../../package.json";

import { defaultLogger } from "../index";

import type { Middleware, MiddlewareRequest, MiddlewareResponse } from "@redwoodjs/vite/middleware";
import type { ApiKeyMiddlewareConfig } from "./types";

const createApiKeyMiddleware = (config: ApiKeyMiddlewareConfig): Middleware => {
  const logger = config.logger || defaultLogger;

  return async (req: MiddlewareRequest, res: MiddlewareResponse) => {
    /**
     * Get key from request and return a response early if not found
     */
    const key = config.getKey
      ? config.getKey(req)
      : req.headers.get("authorization")?.replace("Bearer ", "") ?? null;

    if (key === null) {
      res.body = "Unauthorized";
      res.status = 401;
      return res;
    }

    if (typeof key !== "string") {
      // why not error?
      return key;
    }

    const unkey = new Unkey({
      rootKey: "public",
      wrapperSdkVersion: `@unkey/redwoodjs@${version}`,
      disableTelemetry: config.disableTelemetry,
    });

    const unkeyVerificationResult = await unkey.keys.verify(
      config.apiId ? { key, apiId: config.apiId } : { key },
    );

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

    if (!unkeyVerificationResult.result?.valid) {
      if (config.onInvalidKey) {
        return config.onInvalidKey(req, unkeyVerificationResult.result);
      }

      res.body = "Unauthorized";
      res.status = 401;
    }

    return res;
  };
};

export default createApiKeyMiddleware;
