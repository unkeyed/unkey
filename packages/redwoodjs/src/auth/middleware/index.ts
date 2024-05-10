import { Unkey } from "@unkey/api";
import { version } from "../../../package.json";

import type { Logger } from "@redwoodjs/api/logger";

import type { MiddlewareRequest, MiddlewareResponse } from "@redwoodjs/vite/middleware";
import type { withUnkeyConfig } from "./types";

const defaultLogger = require("abstract-logging") as Logger;

const withUnkey = (config: withUnkeyConfig) => {
  const logger = config.logger || defaultLogger;
  logger.debug("withUnkey middleware");

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
        // why not error?
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
          return config.auth.onError(req, unkeyVerificationResult.error);
        }
        console.error(
          `unkey error: [CODE: ${unkeyVerificationResult.error.code}] - [TRACE: ${unkeyVerificationResult.error.requestId}] - ${unkeyVerificationResult.error.message} - read more at ${unkeyVerificationResult.error.docs}`,
        );

        res.body = "Internal Server Error";
        res.status = 500;
      }

      if (!unkeyVerificationResult.result?.valid) {
        if (config.auth?.onInvalidKey) {
          return config.auth.onInvalidKey(req, unkeyVerificationResult.result);
        }

        res.body = "Unauthorized";
        res.status = 401;
      }

      return res;
    };
  }
};

export default withUnkey;
