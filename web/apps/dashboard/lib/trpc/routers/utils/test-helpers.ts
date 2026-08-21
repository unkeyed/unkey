import { TRPCError } from "@trpc/server";

/**
 * Runs fn and returns what it throws. Fails the test if it does not throw.
 */
function caught(fn: () => unknown): unknown {
  try {
    fn();
  } catch (error) {
    return error;
  }
  throw new Error("expected the call to throw");
}

/**
 * Asserts the value is a TRPCError and narrows it for further assertions.
 */
function asTRPCError(value: unknown): TRPCError {
  if (!(value instanceof TRPCError)) {
    throw new Error(`Expected a TRPCError, got: ${String(value)}`);
  }
  return value;
}

/**
 * Runs fn, which is expected to throw, and returns the thrown TRPCError.
 * Fails the test if fn does not throw or throws something else.
 */
export function catchTrpcError(fn: () => unknown): TRPCError {
  return asTRPCError(caught(fn));
}

/**
 * Awaits the promise's rejection and narrows it to a TRPCError.
 */
export async function rejection(promise: Promise<unknown>): Promise<TRPCError> {
  return asTRPCError(await promise.catch((err: unknown) => err));
}

/**
 * Builds an error shaped like a real mysql2 server error: `message` IS the
 * server's text, so bound values the server quotes back appear in it. The
 * errno/sqlState defaults are ER_DUP_ENTRY's; pass the matching pair for a
 * different server error.
 */
export function mysqlServerError(
  serverText: string,
  code: string,
  errno = 1062,
  sqlState = "23000",
) {
  return Object.assign(new Error(serverText), {
    code,
    errno,
    sqlState,
    sqlMessage: serverText,
  });
}
