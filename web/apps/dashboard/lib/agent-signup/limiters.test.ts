// @vitest-environment node
import { beforeEach, describe, expect, it, vi } from "vitest";

const state = vi.hoisted(() => ({
  rootKey: undefined as string | undefined,
}));

vi.mock("@/lib/env", () => ({
  env: () => ({ UNKEY_ROOT_KEY: state.rootKey }),
}));

import {
  agentLimiters,
  enforceAgentLimit,
  inMemoryAgentLimiterAllowed,
  resetAgentLimitersForTests,
} from "./limiters";

describe("inMemoryAgentLimiterAllowed", () => {
  it("opts in only for local development without VERCEL_ENV", () => {
    expect(inMemoryAgentLimiterAllowed("development", undefined)).toBe(true);
    expect(inMemoryAgentLimiterAllowed("development", "")).toBe(true);
    expect(inMemoryAgentLimiterAllowed("development", "development")).toBe(false);
    expect(inMemoryAgentLimiterAllowed("development", "preview")).toBe(false);
    expect(inMemoryAgentLimiterAllowed("development", "production")).toBe(false);
    expect(inMemoryAgentLimiterAllowed("test", undefined)).toBe(false);
    expect(inMemoryAgentLimiterAllowed("production", undefined)).toBe(false);
  });
});

function setProcessEnv(name: "NODE_ENV" | "VERCEL_ENV", value: string | undefined) {
  const env = process.env as Record<string, string | undefined>;
  if (value === undefined) {
    delete env[name];
    return;
  }
  env[name] = value;
}

describe("agentLimiters", () => {
  const previousNodeEnv = process.env.NODE_ENV;
  const previousVercelEnv = process.env.VERCEL_ENV;

  beforeEach(() => {
    resetAgentLimitersForTests();
    state.rootKey = undefined;
    setProcessEnv("NODE_ENV", previousNodeEnv);
    setProcessEnv("VERCEL_ENV", previousVercelEnv);
  });

  it("fails closed when VERCEL_ENV is unset outside local development", () => {
    setProcessEnv("NODE_ENV", "production");
    setProcessEnv("VERCEL_ENV", undefined);
    expect(() => agentLimiters()).toThrowError(
      expect.objectContaining({ status: 503, code: "rate_limit_unavailable" }),
    );
  });

  it("fails closed when VERCEL_ENV is development but the prefault would have allowed memory", () => {
    setProcessEnv("NODE_ENV", "development");
    setProcessEnv("VERCEL_ENV", "development");
    expect(() => agentLimiters()).toThrowError(
      expect.objectContaining({ status: 503, code: "rate_limit_unavailable" }),
    );
  });

  it("uses the in-memory limiter for explicit local development", async () => {
    setProcessEnv("NODE_ENV", "development");
    setProcessEnv("VERCEL_ENV", undefined);
    const limiters = agentLimiters();
    await expect(enforceAgentLimit(limiters.ip, "127.0.0.1")).resolves.toBeUndefined();
  });

  it("uses the Unkey limiter when a root key is set outside local development", () => {
    setProcessEnv("NODE_ENV", "production");
    setProcessEnv("VERCEL_ENV", "production");
    state.rootKey = "unkey_root";
    expect(() => agentLimiters()).not.toThrow();
  });
});
