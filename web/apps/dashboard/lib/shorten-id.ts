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

  const cut = id.indexOf("_");
  const [prefix, rest] = cut === -1 ? [null, id] : [id.slice(0, cut), id.slice(cut + 1)];

  if (rest.length <= minLength || startChars + endChars >= rest.length) {
    return id;
  }
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
