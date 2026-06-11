/**
 * Low-level url assembly shared by link and route builders. Everything is written
 * verbatim (no encoding) so urls stay human-readable in the address bar; callers
 * must pass url-safe values (ids, enums, slugs, timestamps), never free text.
 *
 */
export type QueryValue = string | number | boolean | null | undefined;
export type QueryParams = Record<string, QueryValue>;
export type BuildUrlParts = {
  base?: string;
  segments?: (string | number)[];
  query?: QueryParams;
};

/** Build a query string from raw values. Null and undefined values are dropped. */
export function toQueryString(query: QueryParams): string {
  const pairs: string[] = [];
  for (const [key, value] of Object.entries(query)) {
    if (value != null) {
      pairs.push(`${key}=${value}`);
    }
  }
  return pairs.join("&");
}

/** Append a query string to an existing base path or url. */
export function withQuery(base: string, query: QueryParams): string {
  const qs = toQueryString(query);
  return qs ? `${base}?${qs}` : base;
}

/**
 * Assemble a url from an optional base and path segments joined with "/". A slash
 * inside a segment ("acme/api") survives verbatim since nothing is encoded.
 */
export function buildUrl(parts: BuildUrlParts): string {
  const path = (parts.segments ?? []).join("/");
  const base = parts.base ?? "";
  const joined = base && path ? `${base}/${path}` : `${base}${path}`;
  return parts.query ? withQuery(joined, parts.query) : joined;
}
