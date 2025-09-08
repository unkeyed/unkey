/**
 * Parses timestamp values and converts microsecond timestamps to milliseconds
 * for proper JavaScript Date construction.
 *
 * This helper detects timestamp precision based on digit count and converts
 * to milliseconds: nanoseconds (>=19 digits) ÷ 1e6, microseconds (>=16 digits) ÷ 1e3,
 * milliseconds (>=13 digits) unchanged, seconds (<13 digits) × 1e3.
 *
 * @param timestamp - The timestamp value as string, number, or Date
 * @returns The timestamp in milliseconds (epoch time)
 *
 * @example
 * // Millisecond timestamp (13 digits) - no conversion needed
 * parseTimestamp(1640995200000) // returns 1640995200000
 *
 * @example
 * // Nanosecond timestamp (19 digits) - converted to milliseconds
 * parseTimestamp(1640995200000000000) // returns 1640995200000
 *
 * @example
 * // Microsecond timestamp (16 digits) - converted to milliseconds
 * parseTimestamp(1640995200000000) // returns 1640995200000
 *
 * @example
 * // Second timestamp (10 digits) - converted to milliseconds
 * parseTimestamp(1640995200) // returns 1640995200000
 *
 * @example
 * // String microsecond timestamp
 * parseTimestamp("1640995200000000") // returns 1640995200000
 *
 * @example
 * // Date object
 * parseTimestamp(new Date(2022, 0, 1)) // returns 1641013200000
 */
export function parseTimestamp(timestamp: string | number | Date): number {
  // Handle null/undefined early
  if (timestamp == null) {
    return 0;
  }

  // Parse timestamp to number with robust validation
  let timestampNum: number;
  if (typeof timestamp === "string") {
    const trimmed = timestamp.trim();
    if (!trimmed) {
      return 0;
    }
    timestampNum = Number.parseFloat(trimmed);
  } else if (timestamp instanceof Date) {
    timestampNum = timestamp.getTime();
  } else {
    timestampNum = timestamp;
  }

  // Validate parsed number
  if (!Number.isFinite(timestampNum)) {
    return 0;
  }

  // Handle sign for negative timestamps
  const isNegative = timestampNum < 0;
  const absoluteTimestamp = Math.abs(timestampNum);

  // Convert to milliseconds based on numeric magnitude thresholds
  let timestampMs: number;
  if (absoluteTimestamp >= 1e18) {
    // Nanoseconds - divide by 1e6 to get milliseconds
    timestampMs = Math.floor(absoluteTimestamp / 1e6);
  } else if (absoluteTimestamp >= 1e15) {
    // Microseconds - divide by 1e3 to get milliseconds
    timestampMs = Math.floor(absoluteTimestamp / 1e3);
  } else if (absoluteTimestamp >= 1e12) {
    // Milliseconds - no scaling needed
    timestampMs = absoluteTimestamp;
  } else {
    // Seconds - multiply by 1e3 to get milliseconds
    timestampMs = absoluteTimestamp * 1e3;
  }

  // Restore sign after conversion
  if (isNegative) {
    timestampMs = -timestampMs;
  }

  return timestampMs;
}
