import { describe, expect, it } from "vitest";
import { createMemoryLimiter } from "./ratelimit";

describe("createMemoryLimiter", () => {
  it("limits one identity without blocking another, and limits an IP separately", async () => {
    let now = 1_000;
    const identity = createMemoryLimiter(2, 60_000, () => now);
    const ip = createMemoryLimiter(2, 60_000, () => now);

    expect((await identity.limit("agent_reg_a")).success).toBe(true);
    expect((await identity.limit("agent_reg_b")).success).toBe(true);
    expect((await identity.limit("agent_reg_a")).success).toBe(true);
    expect((await identity.limit("agent_reg_a")).success).toBe(false);
    expect((await identity.limit("agent_reg_b")).success).toBe(true);

    expect((await ip.limit("203.0.113.10")).success).toBe(true);
    expect((await ip.limit("203.0.113.11")).success).toBe(true);
    expect((await ip.limit("203.0.113.10")).success).toBe(true);
    expect((await ip.limit("203.0.113.10")).success).toBe(false);

    now += 60_000;
    expect((await identity.limit("agent_reg_a")).success).toBe(true);
    expect((await ip.limit("203.0.113.10")).success).toBe(true);
  });
});
