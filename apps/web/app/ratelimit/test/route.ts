import { env } from "@/lib/env";
import { Ratelimit as UnkeyRatelimit } from "@unkey/ratelimit";
import { Ratelimit as UpstashRatelimit } from "@upstash/ratelimit";
import { Redis } from "@upstash/redis";
import { cookies } from "next/headers";
import { z } from "zod";

export const runtime = "edge";

const UNKEY_RATELIMIT_COOKIE = "UNKEY_RATELIMIT";

export const POST = async (req: Request): Promise<Response> => {
  const { limit, duration } = z
    .object({
      limit: z.number().int(),
      duration: z.enum(["1s", "10s", "60s", "5m"]),
    })
    .parse(await req.json());

  const unkeySync = new UnkeyRatelimit({
    namespace: "ratelimit-demo",
    rootKey: env().RATELIMIT_DEMO_ROOT_KEY!,
    limit,
    duration,
    async: false,
  });
  const unkeyAsync = new UnkeyRatelimit({
    namespace: "ratelimit-demo",
    rootKey: env().RATELIMIT_DEMO_ROOT_KEY!,
    limit,
    duration,
    async: true,
  });
  const upstash = new UpstashRatelimit({
    redis: Redis.fromEnv(),
    limiter: UpstashRatelimit.fixedWindow(limit, duration),
  });

  let id: string = crypto.randomUUID();
  const c = cookies().get(UNKEY_RATELIMIT_COOKIE);
  if (c) {
    id = c.value;
  } else {
    cookies().set(UNKEY_RATELIMIT_COOKIE, id, {
      maxAge: 60 * 60 * 24,
    });
  }

  const t1 = performance.now();
  const [unkeySyncResponse, unkeyAsyncResponse, upstashResponse] = await Promise.all([
    unkeySync
      .limit(`${id}-unkey-sync`)
      .then((res) => ({ ...res, latency: performance.now() - t1 })),
    unkeyAsync
      .limit(`${id}-unkey-async`)
      .then((res) => ({ ...res, latency: performance.now() - t1 })),
    upstash.limit(id).then((res) => ({ ...res, latency: performance.now() - t1 })),
  ]);
  console.log({ unkeyAsyncResponse, unkeySyncResponse, upstashResponse });

  return Response.json({
    time: Date.now(),
    unkeySync: unkeySyncResponse,
    unkeyAsync: unkeyAsyncResponse,
    upstash: upstashResponse,
  });
};
