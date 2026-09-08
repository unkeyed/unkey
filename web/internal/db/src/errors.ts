/** See: https://dev.mysql.com/doc/mysql-errors/8.0/en/server-error-reference.html */

const ERR_DEADLOCK = 1213;
const ERR_LOCK_WAIT_TIMEOUT = 1205;
const ERR_TOO_MANY_CONNECTIONS = 1040;
const ERR_SERVER_GONE = 2006;
const ERR_SERVER_LOST = 2013;
const ERR_DUPLICATE_KEY = 1062;

/** mysql2 reports socket failures with a Node `code` and no MySQL error number. */
const CONNECTION_ERROR_CODES = new Set([
  "PROTOCOL_CONNECTION_LOST",
  "PROTOCOL_ENQUEUE_AFTER_FATAL_ERROR",
  "PROTOCOL_ENQUEUE_AFTER_QUIT",
  "PROTOCOL_SEQUENCE_TIMEOUT",
  "POOL_CLOSED",
  "ECONNRESET",
  "ECONNREFUSED",
  "EPIPE",
  "ETIMEDOUT",
  "EHOSTUNREACH",
  "ENETUNREACH",
  "ENOTFOUND",
  "EAI_AGAIN",
]);

type DriverError = {
  code?: unknown;
  errno?: unknown;
};

/** Bounded so a cyclic cause chain cannot hang the caller. */
const MAX_CAUSE_DEPTH = 10;

/** Walks the `cause` chain: drizzle nests the mysql2 error inside a `DrizzleQueryError`. */
function matchesDriverError(err: unknown, match: (driverError: DriverError) => boolean): boolean {
  let current: unknown = err;

  for (let depth = 0; current != null && depth < MAX_CAUSE_DEPTH; depth++) {
    if (typeof current !== "object") {
      return false;
    }

    const candidate = current as DriverError & { cause?: unknown };
    if (match(candidate)) {
      return true;
    }

    current = candidate.cause;
  }

  return false;
}

function matches(err: unknown, errno: number, code: string): boolean {
  return matchesDriverError(err, (e) => e.errno === errno || e.code === code);
}

/** Permanent: the same write fails the same way on a retry. */
export function isDuplicateKeyError(err: unknown): boolean {
  return matches(err, ERR_DUPLICATE_KEY, "ER_DUP_ENTRY");
}

/** InnoDB rolls the whole transaction back, so retry from `BEGIN`. */
export function isDeadlockError(err: unknown): boolean {
  return matches(err, ERR_DEADLOCK, "ER_LOCK_DEADLOCK");
}

export function isLockWaitTimeoutError(err: unknown): boolean {
  return matches(err, ERR_LOCK_WAIT_TIMEOUT, "ER_LOCK_WAIT_TIMEOUT");
}

export function isTooManyConnectionsError(err: unknown): boolean {
  return matches(err, ERR_TOO_MANY_CONNECTIONS, "ER_CON_COUNT_ERROR");
}

/** Server gone, server lost, or a socket failure. A retry gets a new connection. */
export function isConnectionError(err: unknown): boolean {
  return matchesDriverError(err, (driverError) => {
    if (driverError.errno === ERR_SERVER_GONE || driverError.errno === ERR_SERVER_LOST) {
      return true;
    }

    return typeof driverError.code === "string" && CONNECTION_ERROR_CODES.has(driverError.code);
  });
}

export function isTransientError(err: unknown): boolean {
  return (
    isDeadlockError(err) ||
    isLockWaitTimeoutError(err) ||
    isConnectionError(err) ||
    isTooManyConnectionsError(err)
  );
}
