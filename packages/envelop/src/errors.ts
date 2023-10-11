import { createGraphQLError } from "@graphql-tools/utils";
import { UnkeyError } from "@unkey/api";
import type { GraphQLError } from "graphql";
import { UnkeyResult } from "./plugin";

export const UNKEY_ERROR = "Network error!";
export const RATE_LIMIT_ERROR = "Rate limit exceeded!";

export interface HttpGraphQLError extends GraphQLError {
  extensions: {
    http: {
      status?: number;
      headers: Record<string, string>;
    };
  };
}

export const unkeyErrorCodesToStatus = ({
  error,
}: {
  error: typeof UnkeyError;
}): number => {
  let status: number;

  switch (error.code) {
    case "NOT_FOUND":
      status = 404; // Not Found
      break;
    case "BAD_REQUEST":
      status = 400; // Bad Request
      break;
    case "UNAUTHORIZED":
      status = 401; // Unauthorized
      break;
    case "INTERNAL_SERVER_ERROR":
      status = 500; // Internal Server Error
      break;
    case "RATELIMITED":
      status = 429; // Too Many Requests (Rate Limited)
      break;
    case "FORBIDDEN":
      status = 403; // Forbidden
      break;
    case "KEY_USAGE_EXCEEDED":
      status = 429; // Too Many Requests (Key Usage Exceeded)
      break;
    case "INVALID_KEY_TYPE":
      status = 400; // Bad Request (Invalid Key Type)
      break;
    case "NOT_UNIQUE":
      status = 409; // Conflict (Not Unique)
      break;
    case "FETCH_ERROR":
      status = 500; // Internal Server Error (Fetch Error)
      break;
    default:
      status = 500; // Internal Server Error (default case)
  }

  console.debug(`Error Code: ${error.code}, HTTP Status: ${status}`);

  return status;
};

export const createUnkeyError = ({
  errorResponse,
}: {
  errorResponse: UnkeyResult;
}): GraphQLError => {
  // a link to our docs will be in the `error.docs` field
  console.error(UNKEY_ERROR, errorResponse);

  return createGraphQLError(errorResponse.error.message, {
    // path: [fieldNode.alias?.value || fieldDef.name],
    extensions: {
      http: {
        status: unkeyErrorCodesToStatus({ error: errorResponse.error }),
        headers: {},
      },
    },
  });
};

export const createRateLimitError = ({
  result,
}: {
  result: UnkeyResult;
}): GraphQLError => {
  const retryAfter =
    (result.ratelimit?.reset && new Date(result.ratelimit.reset).toISOString()) || "";

  console.warn(RATE_LIMIT_ERROR, result);

  return createGraphQLError(RATE_LIMIT_ERROR, {
    // path: [fieldNode.alias?.value || fieldDef.name],
    extensions: {
      http: {
        status: 429,
        headers: {
          "Retry-After": retryAfter,
        },
      },
    },
  });
};
