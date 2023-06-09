"use server"
import { Redis } from "@upstash/redis"


const redis = Redis.fromEnv()

export async function addToWaitlist(email: string): Promise<number> {
  const key = "waitlist"

  const res = await redis.pipeline().zadd(key, { score: Date.now(), member: email }).zcard(key).exec()
  return res[1]
}
