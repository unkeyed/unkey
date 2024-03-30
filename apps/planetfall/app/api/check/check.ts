import { checkRequestSchema, type checkResponseSchema } from "@/app/lib/schema";
import { verifyKey } from "@unkey/api";
import type { z } from "zod";
export async function POST(request: Request) {
  const key = request.headers.get("Authorization")?.replace("Bearer ", "");
  if (!key) {
    return new Response("unauthorized", { status: 401 });
  }
  const { result, error } = await verifyKey({
    apiId: process.env.UNKEY_API_ID!,
    key,
  });
  if (error) {
    return new Response(error.message, { status: 500 });
  }
  if (!result.valid) {
    return new Response("unauthorized", { status: 403 });
  }

  const req = checkRequestSchema.safeParse(await request.json());
  if (!req.success) {
    return new Response(req.error.message, { status: 400 });
  }

  const checks: z.infer<typeof checkResponseSchema>["checks"] = [];
  for (let i = 0; i < req.data.n; i++) {
    const time = Date.now();
    const res = await fetch(req.data.url, {
      method: req.data.method,
      headers: req.data.headers,
      body: req.data.body,
    }).catch((err) => {
      console.error(err);
      return { status: -1 };
    });

    checks.push({ status: res.status, latency: Date.now() - time });
  }
  return Response.json({ checks });
}
