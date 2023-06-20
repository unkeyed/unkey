"use server";
import { Redis } from "@upstash/redis";
import { headers } from "next/headers";
import { Ratelimit } from "@upstash/ratelimit";
const redis = Redis.fromEnv();
const ratelimit = new Ratelimit({
  redis,
  limiter: Ratelimit.fixedWindow(5, "1m"),
});

export async function addToWaitlist(email: string): Promise<number> {
  const identifier = headers().get("x-real-ip") ?? "global";
  console.log({ identifier });

  const limit = await ratelimit.limit(identifier);
  if (!limit.success) {
    throw new Error("Too many requests, try again later");
  }

  const key = "waitlist";

  const res = await redis
    .pipeline()
    .zadd(key, {
      score: Date.now(),
      member: email,
    })
    .zcard(key)
    .exec();
  return res[1];
}
