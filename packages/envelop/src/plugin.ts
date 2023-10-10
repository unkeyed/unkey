import { Plugin } from "@envelop/core";
import { verifyKey } from "@unkey/api";
import { NetworkError, RateLimitError } from "./errors";

export type UnkeyPluginOptions = {
  token: string;
};

/**
 * A plugin that verifies the Unkey token before executing the query.
 */
export const useUnkey = <TOptions extends UnkeyPluginOptions>(
  options: TOptions
): Plugin => {
  return {
    async onContextBuilding({
      context,
      extendContext,
    }: {
      context: Record<string, any>;
      extendContext: (context: Record<string, any>) => void;
    }) {
      console.debug("Verifying key...", options.token);
      const { result, error: errorResponse } = await verifyKey(options.token);

      if (errorResponse) {
        // handle potential network or bad request error

        // Possible errors codes:
        // "NOT_FOUND"
        // "BAD_REQUEST"
        // "UNAUTHORIZED"
        // "INTERNAL_SERVER_ERROR"
        // "RATELIMITED"
        // "FORBIDDEN"
        // "KEY_USAGE_EXCEEDED"
        // "INVALID_KEY_TYPE"
        // "NOT_UNIQUE"
        // "FETCH_ERROR"; // not from unkey but returned when fetch fails

        // a link to our docs will be in the `error.docs` field
        console.error(errorResponse.error.message, { ...errorResponse.error });
        throw new NetworkError();
      }

      if (!result.valid) {
        console.warn("Rate limit exceeded", result);
        throw new RateLimitError();
      }
      extendContext({
        _unkey: result,
      });
    },
  };
};
