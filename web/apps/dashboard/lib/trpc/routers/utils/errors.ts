import { unwrapDrizzleQueryError } from "@/lib/utils/db-errors";
import { TRPCError } from "@trpc/server";
import { DrizzleQueryError } from "@unkey/db";

/**
 * Re-throws a TRPCError unchanged so meaningful codes reach the client;
 * wraps anything else in a generic INTERNAL_SERVER_ERROR with the original
 * attached as `cause`. The explicit `never` return lets TypeScript treat a
 * call as terminal.
 */
export function wrapUnexpectedError(err: unknown, message: string): never {
  if (err instanceof TRPCError) {
    throw err;
  }
  throw new TRPCError({
    code: "INTERNAL_SERVER_ERROR",
    message,
    cause: err,
  });
}

/**
 * Returns a description of an error that is safe to log. Database and
 * provider errors never contribute their message, which can embed the failed
 * statement, bound values, or user input; only symbolic codes are logged.
 */
export function errorLogDetail(err: unknown): string {
  // Auth-client failures are plain `{success, code, message}` objects whose
  // message may quote user input. Checked by shape before any `instanceof`
  // branch: TRPCError coerces a non-Error `cause` into a real Error with
  // every key (including the raw message) copied over, and that copy must be
  // caught too.
  if (typeof err === "object" && err !== null && "success" in err && err.success === false) {
    const code = stringCode(err);
    if (code !== undefined) {
      return `error code: ${code}`;
    }
    return "auth error: no code";
  }
  if (!(err instanceof Error)) {
    const code = typeof err === "object" && err !== null ? stringCode(err) : undefined;
    if (code !== undefined) {
      return `error code: ${code}`;
    }
    return String(err);
  }
  if (err instanceof DrizzleQueryError) {
    const driverError = unwrapDrizzleQueryError(err);
    // The instanceof re-check stops infinite recursion on a cyclic cause
    // chain.
    if (driverError === undefined || driverError instanceof DrizzleQueryError) {
      return "database query failed";
    }
    return errorLogDetail(driverError);
  }
  if ("sqlState" in err || "sqlMessage" in err) {
    return `database error: ${stringCode(err) ?? err.name}`;
  }
  return err.message;
}

/**
 * Returns the error's `code` property when it is a string, the shape shared
 * by auth-client failures, Node system errors, and mysql2 server errors.
 */
function stringCode(err: object): string | undefined {
  if ("code" in err && typeof err.code === "string") {
    return err.code;
  }
  return undefined;
}
