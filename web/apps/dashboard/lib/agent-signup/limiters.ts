import { env } from "@/lib/env";
import { Ratelimit } from "@unkey/ratelimit";
import { AgentSignupError } from "./errors";
import {
  AGENT_IDENTITY_LIMIT,
  AGENT_IP_LIMIT,
  AGENT_LIMIT_WINDOW_MS,
  type Limiter,
  createMemoryLimiter,
} from "./ratelimit";

let cached: { identity: Limiter; ip: Limiter } | undefined;

function unkeyLimiter(rootKey: string, namespace: string, limit: number): Limiter {
  return new Ratelimit({
    rootKey,
    namespace,
    limit,
    duration: "60s",
  });
}

// env().VERCEL_ENV prefaults an absent value to "development", which would
// select the in-memory limiter in any environment that forgot the variable.
// Read the process environment so only an explicit local dev process opts in.
export function inMemoryAgentLimiterAllowed(
  nodeEnv: string | undefined = process.env.NODE_ENV,
  vercelEnv: string | undefined = process.env.VERCEL_ENV,
): boolean {
  return nodeEnv === "development" && (vercelEnv === undefined || vercelEnv === "");
}

export function resetAgentLimitersForTests(): void {
  cached = undefined;
}

export function agentLimiters(): { identity: Limiter; ip: Limiter } {
  if (cached) {
    return cached;
  }
  const rootKey = env().UNKEY_ROOT_KEY;
  if (rootKey) {
    cached = {
      identity: unkeyLimiter(rootKey, "agent_signup_identity", AGENT_IDENTITY_LIMIT),
      ip: unkeyLimiter(rootKey, "agent_signup_ip", AGENT_IP_LIMIT),
    };
    return cached;
  }
  if (!inMemoryAgentLimiterAllowed()) {
    throw new AgentSignupError(
      503,
      "rate_limit_unavailable",
      "Agent signup rate limiting is not configured.",
    );
  }
  cached = {
    identity: createMemoryLimiter(AGENT_IDENTITY_LIMIT, AGENT_LIMIT_WINDOW_MS),
    ip: createMemoryLimiter(AGENT_IP_LIMIT, AGENT_LIMIT_WINDOW_MS),
  };
  return cached;
}

export async function enforceAgentLimit(limiter: Limiter, identifier: string): Promise<void> {
  let decision: { success: boolean };
  try {
    decision = await limiter.limit(identifier);
  } catch {
    throw new AgentSignupError(
      503,
      "rate_limit_unavailable",
      "Agent signup rate limiting is unavailable.",
    );
  }
  if (!decision.success) {
    throw new AgentSignupError(429, "rate_limited", "Too many requests. Try again later.");
  }
}
