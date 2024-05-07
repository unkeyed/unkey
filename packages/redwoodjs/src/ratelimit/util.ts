import type { MiddlewareRequest } from "@redwoodjs/vite/middleware";
import { MiddlewareResponse } from "@redwoodjs/vite/middleware";

/**
 * defaultRatelimitIdentifier is the default function to generate a ratelimit identifier
 *
 * Use the req serverAuthContext to access authentication information, if available
 * in order to access a current user identifier
 *
 */
export const defaultRatelimitIdentifier = (req: MiddlewareRequest) => {
  const authContext = req?.serverAuthContext?.get();
  const identifier = authContext?.isAuthenticated
    ? Buffer.from(JSON.stringify(authContext.currentUser)).toString("base64")
    : "192.168.1.1";
  return identifier;
};

/*
 * defaultRatelimitExceededResponse is the default response when the rate limit is exceeded
 */
export const defaultRatelimitExceededResponse = (_req: MiddlewareRequest) => {
  return new MiddlewareResponse("Rate limit exceeded", { status: 429 });
};

/**
 * defaultRatelimitErrorResponse is the default response when there is an error with the rate limit
 */
export const defaultRatelimitErrorResponse = (_req: MiddlewareRequest) => {
  return new MiddlewareResponse("Internal server error", { status: 500 });
};
