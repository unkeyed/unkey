import { env } from "@/lib/env";
import { newId } from "@unkey/id";
import { Ratelimit } from "@unkey/ratelimit";
import { cookies } from "next/headers";
import { z } from "zod";

const UNKEY_RATELIMIT_COOKIE = "UNKEY_RATELIMIT";

export const POST = async (req: Request): Promise<Response> => {
  const { limit, duration } = z
    .object({
      limit: z.number().int(),
      duration: z.enum(["1s", "10s", "60s", "5m"]),
    })
    .parse(await req.json());

  const rl = new Ratelimit({
    namespace: "ratelimit-demo",
    rootKey: env().RATELIMIT_DEMO_ROOT_KEY!,
    limit,
    duration,
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
