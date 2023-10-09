import { Plugin } from "@envelop/core";
import { type UnkeyError, verifyKey } from "@unkey/api";

export type UnkeyPluginOptions = { token: string; logger?: any };

/**
 * export type ErrorCode =
  | "NOT_FOUND"
  | "BAD_REQUEST"
  | "UNAUTHORIZED"
  | "INTERNAL_SERVER_ERROR"
  | "RATELIMITED"
  | "FORBIDDEN"
  | "KEY_USAGE_EXCEEDED"
  | "INVALID_KEY_TYPE"
  | "NOT_UNIQUE"
  | "FETCH_ERROR"; // not from unkey but returned when fetch fails

 */
export class NetworkError extends Error {}
export class RateLimitError extends Error {}

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
      try {
        const { result, error } = await verifyKey(options.token);

        if (error) {
          // handle potential network or bad request error
          // a link to our docs will be in the `error.docs` field
          console.error(error.message, error.code, error.requestId);
          throw new NetworkError(`Network error!`);
        }

        if (!result.valid) {
          throw new RateLimitError(`Rate limit exceeded!`);
        }
        extendContext({
          ["_unkey"]: result,
        });
      } catch (e) {
        throw e;
      }
    },
  };
};
