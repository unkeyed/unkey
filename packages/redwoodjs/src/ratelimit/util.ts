import { type MatchFunction, match } from "path-to-regexp";

import type { MiddlewareRequest } from "@redwoodjs/vite/middleware";
import { MiddlewareResponse } from "@redwoodjs/vite/middleware";

export type MiddlewarePathMatcher = string | string[];

export const defaultRatelimitIdentifier = (req: MiddlewareRequest) => {
  const authContext = req?.serverAuthContext?.get();
  const identifier = authContext?.isAuthenticated
    ? Buffer.from(JSON.stringify(authContext.currentUser)).toString("base64")
    : "192.168.1.1";
  return identifier;
};

export const matchesPath = (path: string, matcher: MiddlewarePathMatcher): boolean => {
  // Convert matcher to an array if it's not already one
  const matchers = Array.isArray(matcher) ? matcher : [matcher];

  console.debug(">>>> in matchesPath", matchers, path);

  // Create a list of matching functions from the matchers
  const matchingFunctions: MatchFunction[] = matchers.map((pattern) =>
    match(pattern, { decode: decodeURIComponent }),
  );

  // Check if the path matches any of the patterns
  return matchingFunctions.some((matchFunc) => matchFunc(path) !== false);
};

export const defaultRatelimitExceededResponse = (_req: MiddlewareRequest) => {
  return new MiddlewareResponse("Rate limit exceeded", { status: 429 });
};

export const defaultRatelimitErrorResponse = (_req: MiddlewareRequest) => {
  return new MiddlewareResponse("Internal server error", { status: 500 });
};
