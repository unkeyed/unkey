// safeResolve evaluates a feature flag and falls back when the provider fails.
// Local development should not break the dashboard because an optional flag
// backend is unavailable.
export async function safeResolve<T>(
  key: string,
  resolve: () => Promise<T>,
  fallback: T,
): Promise<T> {
  try {
    return await resolve();
  } catch (error) {
    console.warn(`[flags] failed to resolve ${key}, using fallback`, error);
    return fallback;
  }
}
