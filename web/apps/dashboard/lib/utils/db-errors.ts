/** Re-exported from `@unkey/db`, where they live next to the retry helpers that use them. */
export {
  isConnectionError,
  isDeadlockError,
  isDuplicateKeyError,
  isLockWaitTimeoutError,
  isTooManyConnectionsError,
  isTransientError,
} from "@unkey/db";
