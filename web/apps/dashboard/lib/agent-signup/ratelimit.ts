export const AGENT_IDENTITY_LIMIT = 10;
export const AGENT_IP_LIMIT = 30;
export const AGENT_LIMIT_WINDOW_MS = 60_000;

export type LimitDecision = {
  success: boolean;
};

export type Limiter = {
  limit: (identifier: string) => Promise<LimitDecision>;
};

export function createMemoryLimiter(
  limit: number,
  windowMs: number,
  now: () => number = Date.now,
): Limiter {
  const hits = new Map<string, number[]>();
  return {
    async limit(identifier: string) {
      const at = now();
      const recent = (hits.get(identifier) ?? []).filter((ts) => at - ts < windowMs);
      if (recent.length >= limit) {
        hits.set(identifier, recent);
        return { success: false };
      }
      recent.push(at);
      hits.set(identifier, recent);
      return { success: true };
    },
  };
}
