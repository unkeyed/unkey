/**
 * Validates that a redirect URL is a safe relative path.
 * Uses an allowlist approach. Every path segment must contain only
 * alphanumeric characters, hyphens, underscores, dots, or tildes.
 * Query strings and hash fragments are allowed (they stay on the same origin).
 *
 * Lives in lib/auth so server and client routes share one validation boundary.
 */
export function isSafeRedirectPath(url: string): boolean {
  if (!url.startsWith("/") || url.startsWith("//")) {
    return false;
  }

  if (hasUnsafeRedirectCharacter(url)) {
    return false;
  }

  // Encoded separators, dot segments, controls, and a second layer of
  // encoding are rejected before URL parsing can normalize them.
  if (/%(?:00|25|2e|2f|5c)/i.test(url) || /%(?![0-9a-f]{2})/i.test(url)) {
    return false;
  }

  let decoded: string;
  try {
    decoded = decodeURIComponent(url);
  } catch {
    return false;
  }

  if (hasUnsafeRedirectCharacter(decoded)) {
    return false;
  }

  const [pathAndQuery] = url.split("#", 1);
  const queryIndex = pathAndQuery.indexOf("?");
  const pathOnly = queryIndex === -1 ? pathAndQuery : pathAndQuery.slice(0, queryIndex);
  const query = queryIndex === -1 ? "" : pathAndQuery.slice(queryIndex + 1);

  const segments = pathOnly.split("/");
  for (let i = 1; i < segments.length; i++) {
    if (segments[i] === "" && i === segments.length - 1) {
      continue;
    }

    let segment: string;
    try {
      segment = decodeURIComponent(segments[i]);
    } catch {
      return false;
    }

    if (
      !segment ||
      segment === "." ||
      segment === ".." ||
      !/^[a-zA-Z0-9._~!$&'()*+,;=:@-]+$/.test(segment)
    ) {
      return false;
    }
  }

  const queryKeys = new Set<string>();
  for (const key of new URLSearchParams(query).keys()) {
    if (queryKeys.has(key)) {
      return false;
    }
    queryKeys.add(key);
  }

  return true;
}

function hasUnsafeRedirectCharacter(value: string): boolean {
  for (const character of value) {
    const codePoint = character.codePointAt(0) ?? 0;
    if (
      codePoint <= 31 ||
      codePoint === 127 ||
      codePoint === 92 ||
      codePoint === 0x2028 ||
      codePoint === 0x2029 ||
      codePoint === 0x2044 ||
      codePoint === 0x2215
    ) {
      return true;
    }
  }
  return false;
}

/**
 * Sanitizes an untrusted redirect target: returns it only when it is a
 * string that passes isSafeRedirectPath, otherwise the fallback. Takes
 * `unknown` because callers hand it values from JSON.parse of the OAuth
 * `state` param and from client-supplied server-action inputs, where
 * TypeScript types don't hold at runtime.
 */
export function sanitizeRedirectPath(url: unknown, fallback = "/apis"): string {
  return typeof url === "string" && isSafeRedirectPath(url) ? url : fallback;
}

type SignInLocation = Pick<Location, "pathname" | "search" | "assign">;

/**
 * Leaves the mounted application before starting hosted authentication so
 * protected queries cannot continue running during the redirect.
 */
export function redirectToSignIn(location: SignInLocation): void {
  const currentPath = `${location.pathname}${location.search}`;
  const signInUrl =
    currentPath && currentPath !== "/"
      ? `/auth/sign-in?redirect=${encodeURIComponent(currentPath)}`
      : "/auth/sign-in";

  location.assign(signInUrl);
}
