import { newId } from "@unkey/id";
import { Ratelimit } from "@upstash/ratelimit";
import { Redis } from "@upstash/redis";
import { cookies } from "next/headers";
import { z } from "zod";

export const runtime = "edge";

const UNKEY_RATELIMIT_COOKIE = "UNKEY_RATELIMIT_REDIS";

export const POST = async (req: Request): Promise<Response> => {
  const { limit, duration } = z
    .object({
      limit: z.number().int(),
      duration: z.enum(["1s", "10s", "60s", "5m"]),
    })
    .parse(await req.json());

  const rl = new Ratelimit({
    redis: Redis.fromEnv(),
    limiter: Ratelimit.fixedWindow(limit, duration),
  });

  let id: string = newId("ratelimit");
  const c = cookies().get(UNKEY_RATELIMIT_COOKIE);
  if (c) {
    id = c.value;
  } else {
    cookies().set(UNKEY_RATELIMIT_COOKIE, id, {
      maxAge: 60 * 60 * 24,
    });
  }

  const t1 = performance.now();
  const res = await rl.limit(id);
  return Response.json({ ...res, time: Date.now(), latency: performance.now() - t1 });
};
