/**
 * Parses timestamp values and converts microsecond timestamps to milliseconds
 * for proper JavaScript Date construction.
 *
 * This helper detects microsecond precision timestamps (>= 16 digits or > 1e13)
 * and automatically converts them to milliseconds by dividing by 1000.
 *
 * @param timestamp - The timestamp value as string, number, or Date
 * @returns The timestamp in milliseconds (epoch time)
 *
 * @example
 * // Millisecond timestamp (13 digits) - no conversion needed
 * parseTimestamp(1640995200000) // returns 1640995200000
 *
 * @example
 * // Microsecond timestamp (16 digits) - converted to milliseconds
 * parseTimestamp(1640995200000000) // returns 1640995200000
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

  // Compute digit count by converting absolute integer part to string and stripping non-digits
  const integerPart = Math.floor(absoluteTimestamp);
  const digitString = integerPart.toString().replace(/\D/g, "");
  const digitCount = digitString.length;

  // Treat as microseconds when digitCount >= 16 or timestampNum > 1e13
  const isMicroseconds = digitCount >= 16 || absoluteTimestamp > 1e13;

  // Convert microseconds to milliseconds if needed
  let timestampMs = isMicroseconds
    ? Math.floor(absoluteTimestamp / 1000)
    : absoluteTimestamp;

  // Restore sign after conversion
  if (isNegative) {
    timestampMs = -timestampMs;
  }

  return timestampMs;
}
