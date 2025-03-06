import type { KeysOverviewLog } from "@unkey/clickhouse/src/keys/keys";

/**
 * Severity levels for error percentages as string literal union
 */
export type ErrorSeverity = "none" | "low" | "moderate" | "high";

/**
 * Calculate the error percentage for a key and return a numeric value
 * @param log The keys overview log
 * @returns The percentage of errors (0-100)
 */
export const getErrorPercentage = (log: KeysOverviewLog): number => {
  const totalRequests = log.valid_count + log.error_count;

  // Avoid division by zero
  if (totalRequests === 0) {
    return 0;
  }

  return (log.error_count / totalRequests) * 100;
};

/**
 * Determine the error severity based on the error percentage
 * @param log The keys overview log
 * @returns The severity level as a string literal
 */
export const getErrorSeverity = (log: KeysOverviewLog): ErrorSeverity => {
  const errorPercentage = getErrorPercentage(log);

  if (errorPercentage >= 50) {
    return "high";
  }
  if (errorPercentage >= 20) {
    return "moderate";
  }
  if (errorPercentage > 0) {
    return "low";
  }
  return "none";
};

/**
 * Legacy function maintained for backward compatibility
 * Returns true if more than 30% of requests are errors
 * @deprecated Use getErrorPercentage or getErrorSeverity instead
 */
export const calculateErrorPercentage = (log: KeysOverviewLog): boolean => {
  return getErrorPercentage(log) > 30;
};
