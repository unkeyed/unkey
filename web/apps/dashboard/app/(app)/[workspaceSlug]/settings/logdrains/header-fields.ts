/** Matches the RFC 9110 token characters that are valid in an HTTP field name. */
export const headerNamePattern = /^[!#$%&'*+.^_`|~0-9A-Za-z-]+$/;

export function isValidHttpHeaderValue(value: string): boolean {
  for (const character of value) {
    const codePoint = character.codePointAt(0);
    if (codePoint === undefined || (codePoint < 32 && codePoint !== 9) || codePoint === 127) {
      return false;
    }
  }
  return true;
}

export function toHeaderRecord(
  headers: ReadonlyArray<{ name: string; value: string }>,
): Record<string, string> {
  return Object.fromEntries(
    headers
      .filter((header) => header.name.trim() !== "" || header.value !== "")
      .map((header) => [header.name.trim(), header.value]),
  );
}
