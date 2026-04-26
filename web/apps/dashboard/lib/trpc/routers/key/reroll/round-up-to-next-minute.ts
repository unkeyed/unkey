const MINUTE_IN_MS = 60_000;

/**
 * Rounds a timestamp up to the next whole minute. Timestamps that already
 * fall on a minute boundary are returned unchanged. Mirrors the server-side
 * reroll handler, which expires rerolled keys on minute-aligned boundaries
 * to simplify downstream caching.
 */
export function roundUpToNextMinute(date: Date): Date {
  const ms = date.getTime();
  if (ms % MINUTE_IN_MS === 0) {
    return new Date(ms);
  }
  return new Date(Math.ceil(ms / MINUTE_IN_MS) * MINUTE_IN_MS);
}
