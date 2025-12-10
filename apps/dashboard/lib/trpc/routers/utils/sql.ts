/**
 * Escapes special characters in a string for use in SQL LIKE patterns.
 * This prevents user input from being interpreted as wildcards.
 *
 * @param value - The user input to escape
 * @returns The escaped string safe for use in LIKE patterns
 *
 * @example
 * ```ts
 * const userInput = "test_value%";
 * const escaped = escapeLike(userInput); // "test\_value\%"
 * like(column, `%${escaped}%`) // Will match literal underscores and percent signs
 * ```
 */
export function escapeLike(value: string): string {
  return value.replace(/\\/g, "\\\\").replace(/%/g, "\\%").replace(/_/g, "\\_");
}
