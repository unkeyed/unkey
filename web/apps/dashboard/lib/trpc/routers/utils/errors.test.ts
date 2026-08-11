import { TRPCError } from "@trpc/server";
import { DrizzleQueryError } from "@unkey/db";
import { describe, expect, it } from "vitest";
import { errorLogDetail, wrapUnexpectedError } from "./errors";
import { catchTrpcError, mysqlServerError } from "./test-helpers";

describe("wrapUnexpectedError", () => {
  it("re-throws an existing TRPCError unchanged so meaningful codes and messages reach the client", () => {
    const original = new TRPCError({ code: "NOT_FOUND", message: "Checkout session not found" });

    const thrown = catchTrpcError(() => {
      wrapUnexpectedError(original, "generic message");
    });

    expect(thrown).toBe(original);
  });

  it("wraps an unexpected error in INTERNAL_SERVER_ERROR carrying only the generic message, so internal details never reach the client", () => {
    const internal = new Error("connect ECONNREFUSED 127.0.0.1:3306");

    const thrown = catchTrpcError(() => {
      wrapUnexpectedError(internal, "Please try again or contact support@unkey.com");
    });

    expect(thrown.code).toBe("INTERNAL_SERVER_ERROR");
    expect(thrown.message).toBe("Please try again or contact support@unkey.com");
    expect(thrown.message).not.toContain("ECONNREFUSED");
  });

  it("attaches the original error as `cause` so callers can log its safe detail", () => {
    const internal = new Error("connect ECONNREFUSED 127.0.0.1:3306");

    const thrown = catchTrpcError(() => {
      wrapUnexpectedError(internal, "Unable to update workspace");
    });

    expect(thrown.cause).toBe(internal);
  });
});

describe("errorLogDetail", () => {
  it("returns the message of an ordinary error", () => {
    expect(errorLogDetail(new Error("boom"))).toBe("boom");
  });

  it("stringifies a non-Error value", () => {
    expect(errorLogDetail("string failure")).toBe("string failure");
  });

  it("extracts the code from a thrown plain object like the auth client's AuthErrorResponse instead of '[object Object]'", () => {
    const authError = {
      success: false,
      code: "ORG_UPDATE_FAILED",
      message: "Could not update organization 'Acme Corp'",
    };

    const detail = errorLogDetail(authError);

    expect(detail).toBe("error code: ORG_UPDATE_FAILED");
    expect(detail).not.toContain("Acme Corp");
  });

  it("reduces a mysql2 server error to its symbolic code, keeping quoted bound values out of the logs", () => {
    const driverError = mysqlServerError(
      "Duplicate entry 'Acme Corp' for key 'name'",
      "ER_DUP_ENTRY",
    );

    const detail = errorLogDetail(driverError);

    expect(detail).toBe("database error: ER_DUP_ENTRY");
    expect(detail).not.toContain("Acme Corp");
  });

  it("unwraps a DrizzleQueryError, whose own message embeds the failed statement and its params", () => {
    const driverError = mysqlServerError(
      "Duplicate entry 'Acme Corp' for key 'name'",
      "ER_DUP_ENTRY",
    );
    const wrapped = new DrizzleQueryError(
      "update `workspaces` set `name` = ? where `id` = ?",
      ["Acme Corp", "ws_123"],
      driverError,
    );

    const detail = errorLogDetail(wrapped);

    expect(detail).toBe("database error: ER_DUP_ENTRY");
    expect(detail).not.toContain("Acme Corp");
    expect(detail).not.toContain("ws_123");
    expect(detail).not.toContain("workspaces");
  });

  it("describes a DrizzleQueryError without a cause generically rather than using its statement-bearing message", () => {
    const wrapped = new DrizzleQueryError("update `workspaces` set `name` = ? where `id` = ?", [
      "Acme Corp",
      "ws_123",
    ]);

    const detail = errorLogDetail(wrapped);

    expect(detail).toBe("database query failed");
  });

  it("describes a cyclic DrizzleQueryError cause chain generically instead of recursing until the stack overflows", () => {
    const wrapped = new DrizzleQueryError("update `workspaces` set `name` = ? where `id` = ?", [
      "Acme Corp",
    ]);
    wrapped.cause = wrapped;

    const detail = errorLogDetail(wrapped);

    expect(detail).toBe("database query failed");
  });

  it("still extracts only the symbolic code after @trpc/server coerces a plain-object auth error attached as `cause` into a real Error", () => {
    // TRPCError copies every key of a non-Error cause (including the raw
    // message) onto a synthetic Error; the sanitizer must catch that coerced
    // copy, not just the bare object.
    const wrapped = new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "We are unable to update the workspace name",
      cause: {
        success: false,
        code: "ORG_UPDATE_FAILED",
        message: "Could not update organization 'Acme Corp'",
      },
    });

    const detail = errorLogDetail(wrapped.cause);

    expect(detail).toBe("error code: ORG_UPDATE_FAILED");
    expect(detail).not.toContain("Acme Corp");
  });
});
