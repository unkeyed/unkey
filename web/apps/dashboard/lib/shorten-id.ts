/**
 * Shortens an ID by keeping a specified number of characters from the start (after prefix) and end,
 * with customizable separator in between.
 */
export function shortenId(
  id: string,
  options: {
    /** Number of characters to keep from the start (default: 8) */
    startChars?: number;
    /** Number of characters to keep from the end (default: 4) */
    endChars?: number;
    /** Separator between start and end (default: "...") */
    separator?: string;
    /** Minimum length required to apply shortening (default: startChars + endChars + 3) */
    minLength?: number;
  } = {},
): string {
  const {
    startChars = 4,
    endChars = 4,
    separator = "...",
    minLength = startChars + endChars + 3,
  } = options;

  // Validate inputs
  if (startChars < 0 || endChars < 0) {
    throw new Error("startChars and endChars must be non-negative");
  }

  if (startChars + endChars === 0) {
    throw new Error("At least one of startChars or endChars must be greater than 0");
  }

  // Return original if too short to meaningfully shorten
  if (id.length <= minLength) {
    return id;
  }

  // Handle edge case where requested chars exceed ID length
  if (startChars + endChars >= id.length) {
    return id;
  }

  const [prefix, rest] = id.includes("_") ? id.split("_", 2) : [null, id];
  let s = "";
  if (prefix) {
    s += prefix;
    s += "_";
  }
  s += rest.substring(0, startChars);
  s += separator;
  s += rest.substring(rest.length - endChars);
  return s;
}
