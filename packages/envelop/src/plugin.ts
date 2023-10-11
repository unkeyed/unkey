import { Plugin } from "@envelop/core";
import { verifyKey } from "@unkey/api";
import type { GraphQLError } from "graphql";
import { createRateLimitError, createUnkeyError } from "./errors";

const UNKEY_CONTEXT_KEY = "_unkey";

/* eslint-disable @typescript-eslint/no-explicit-any */
export interface UnkeyResult {
  [key: string]: any;
}

export type UnkeyPluginOptions = {
  token: string;
};

/**
 * A plugin that verifies the Unkey token before executing the query.
 */
export const useUnkey = <TOptions extends UnkeyPluginOptions>(
  options: TOptions
): Plugin => {
  const token = options.token;

  return {
    onContextBuilding({
      // context,
      extendContext,
    }: {
      // context: Record<string, any>;
      extendContext: (
        context: Record<string, string | number | boolean>
      ) => void;
    }) {
      // here we  stash the Unkey API token on the context
      // for potential later use in the execute phase

      // We could also extract the token from the headers here
      // for a per user/session authorization check later
      extendContext({
        [UNKEY_CONTEXT_KEY]: token,
      });
    },
    async onExecute({
      //args,
      setResultAndStopExecution,
    }) {
      const errors: GraphQLError[] = [];

      // Note: Can fetch the the stashes session token from the context here
      // with args and contextValue in future
      if (!token) {
        errors.push(
          createUnkeyError({
            errorResponse: {
              error: {
                message: "Missing Unkey Token",
                code: "INVALID_KEY_TYPE",
              },
              result: {
                valid: false,
              },
            },
          })
        );
      }

      // we have a token, so we can verify it
      if (errors.length === 0) {
        console.debug("Verifying key ...", token);

        const { result, error: errorResponse } = await verifyKey(token);

        // handle potential network or bad request error
        if (errorResponse) {
          errors.push(createUnkeyError({ errorResponse }));
        }

        // handle rate limit error
        if (!result.valid) {
          errors.push(createRateLimitError({ result }));
        }
      }

      if (errors.length > 0) {
        setResultAndStopExecution({
          data: null,
          errors,
        });
      }
    },
  };
};
