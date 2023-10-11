/* eslint-disable @typescript-eslint/no-explicit-any */
import { assertSingleExecutionValue } from "@envelop/testing";

import { createRateLimitError, createUnkeyError, unkeyErrorCodesToStatus } from "../errors";

import type { HttpGraphQLError } from "../errors";

export const extractGraphQLHttpError = (result: any): HttpGraphQLError => {
  assertSingleExecutionValue(result);

  expect(result.data).toBeNull();
  expect(result.errors).toBeDefined();
  expect(result.errors).toHaveLength(1);

  const error = result.errors?.[0] as HttpGraphQLError;
  return error;
};

describe("errors", () => {
  describe("Convert Unkey Codes to HTTP Status", () => {
    it('should return 404 for "NOT_FOUND"', () => {
      const error = { code: "NOT_FOUND" };
      expect(unkeyErrorCodesToStatus({ error })).toBe(404);
    });

    it('should return 400 for "BAD_REQUEST"', () => {
      const error = { code: "BAD_REQUEST" };
      expect(unkeyErrorCodesToStatus({ error })).toBe(400);
    });

    it('should return 401 for "UNAUTHORIZED"', () => {
      const error = { code: "UNAUTHORIZED" };
      expect(unkeyErrorCodesToStatus({ error })).toBe(401);
    });

    it('should return 500 for "INTERNAL_SERVER_ERROR"', () => {
      const error = { code: "INTERNAL_SERVER_ERROR" };
      expect(unkeyErrorCodesToStatus({ error })).toBe(500);
    });

    it('should return 429 for "RATELIMITED"', () => {
      const error = { code: "RATELIMITED" };
      expect(unkeyErrorCodesToStatus({ error })).toBe(429);
    });

    it('should return 403 for "FORBIDDEN"', () => {
      const error = { code: "FORBIDDEN" };
      expect(unkeyErrorCodesToStatus({ error })).toBe(403);
    });

    it('should return 429 for "KEY_USAGE_EXCEEDED"', () => {
      const error = { code: "KEY_USAGE_EXCEEDED" };
      expect(unkeyErrorCodesToStatus({ error })).toBe(429);
    });

    it('should return 400 for "INVALID_KEY_TYPE"', () => {
      const error = { code: "INVALID_KEY_TYPE" };
      expect(unkeyErrorCodesToStatus({ error })).toBe(400);
    });

    it('should return 409 for "NOT_UNIQUE"', () => {
      const error = { code: "NOT_UNIQUE" };
      expect(unkeyErrorCodesToStatus({ error })).toBe(409);
    });

    it('should return 500 for "FETCH_ERROR"', () => {
      const error = { code: "FETCH_ERROR" };
      expect(unkeyErrorCodesToStatus({ error })).toBe(500);
    });

    it("should return 500 for an unknown error code", () => {
      const error = { code: "UNKNOWN_ERROR_CODE" };
      expect(unkeyErrorCodesToStatus({ error })).toBe(500);
    });
  });

  describe("Create Unkey errors", () => {
    const checkUnkeyError = (error: HttpGraphQLError, status: number) => {
      expect(error).toHaveProperty("message");
      expect(error).toHaveProperty("extensions.http.headers");
      expect(error.extensions.http.status).toEqual(status);
    };
    it('should create an Unkey Error for "NOT_FOUND"', () => {
      const error = createUnkeyError({
        errorResponse: { error: { code: "NOT_FOUND" } },
      }) as HttpGraphQLError;
      checkUnkeyError(error, 404);
    });

    it('should create an Unkey Error for "BAD_REQUEST"', () => {
      const error = createUnkeyError({
        errorResponse: { error: { code: "BAD_REQUEST" } },
      }) as HttpGraphQLError;

      checkUnkeyError(error, 400);
    });

    it('should create an Unkey Error for "UNAUTHORIZED"', () => {
      const error = createUnkeyError({
        errorResponse: { error: { code: "UNAUTHORIZED" } },
      }) as HttpGraphQLError;

      checkUnkeyError(error, 401);
    });

    it('should create an Unkey Error for "INTERNAL_SERVER_ERROR"', () => {
      const error = createUnkeyError({
        errorResponse: { error: { code: "INTERNAL_SERVER_ERROR" } },
      }) as HttpGraphQLError;

      checkUnkeyError(error, 500);
    });

    it('should create an Unkey Error for "RATELIMITED"', () => {
      const error = createUnkeyError({
        errorResponse: { error: { code: "RATELIMITED" } },
      }) as HttpGraphQLError;

      checkUnkeyError(error, 429);
    });

    it('should create an Unkey Error for "FORBIDDEN"', () => {
      const error = createUnkeyError({
        errorResponse: { error: { code: "FORBIDDEN" } },
      }) as HttpGraphQLError;

      checkUnkeyError(error, 403);
    });

    it('should create an Unkey Error for "KEY_USAGE_EXCEEDED"', () => {
      const error = createUnkeyError({
        errorResponse: { error: { code: "KEY_USAGE_EXCEEDED" } },
      }) as HttpGraphQLError;

      checkUnkeyError(error, 429);
    });

    it('should create an Unkey Error for "INVALID_KEY_TYPE"', () => {
      const error = createUnkeyError({
        errorResponse: { error: { code: "INVALID_KEY_TYPE" } },
      }) as HttpGraphQLError;

      checkUnkeyError(error, 400);
    });

    it('should create an Unkey Error for "NOT_UNIQUE"', () => {
      const error = createUnkeyError({
        errorResponse: { error: { code: "NOT_UNIQUE" } },
      }) as HttpGraphQLError;

      checkUnkeyError(error, 409);
    });

    it('should create an Unkey Error for "FETCH_ERROR"', () => {
      const error = createUnkeyError({
        errorResponse: { error: { code: "FETCH_ERROR" } },
      }) as HttpGraphQLError;

      checkUnkeyError(error, 500);
    });

    it("should create an Unkey Error for an unknown error code", () => {
      const error = createUnkeyError({
        errorResponse: { error: { code: "UNKNOWN_ERROR_CODE" } },
      }) as HttpGraphQLError;

      checkUnkeyError(error, 500);
    });
  });

  describe("Create Rate Limit errors", () => {
    it("should create a Rate Limit Error", () => {
      createRateLimitError({ result: { valid: false } });
    });
  });
});
