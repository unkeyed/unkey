import type { MiddlewareRequest } from "@redwoodjs/vite/middleware";
import { MiddlewareResponse } from "@redwoodjs/vite/middleware";
import { match } from "path-to-regexp";
import type { MatchFunction } from "path-to-regexp";

export type MiddlewarePathMatcher = string | string[];

/**
 * defaultRatelimitIdentifier is the default function to generate a ratelimit identifier
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

/**
 * matchesPath checks if a path matches a pattern
 */
export const matchesPath = (path: string, matcher: MiddlewarePathMatcher): boolean => {
  try {
    // Convert matcher to an array if it's not already one
    const matchers = Array.isArray(matcher) ? matcher : [matcher];

    // Create a list of matching functions from the matchers
    const matchingFunctions: MatchFunction[] = matchers.map((pattern) => {
      return match(pattern, { decode: decodeURIComponent });
    });

    // Check if the path matches any of the patterns
    return matchingFunctions.some((matchFunc) => matchFunc(path) !== false);
  } catch (e) {
    console.error("Error in matchesPath", e);
    throw e;
  }
};
