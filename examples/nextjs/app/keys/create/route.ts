import { NextRequest, NextResponse } from "next/server";
import { Unkey } from "@unkey/api";

export const dynamic = "force-dynamic";
export async function GET(req: NextRequest) {
  const unkey = new Unkey({ token: process.env.UNKEY_TOKEN! });
  const url = new URL(req.url);
  const created = await unkey.keys.create({
    apiId: process.env.UNKEY_API_ID!,
    ratelimit: {
      type: "fast",
      limit: 5,
      refillRate: 1,
      refillInterval: 1000,
    },
    meta: {
      random: Math.random(),
    },
    // ..
  });

  if (created.error) {
    return NextResponse.json({ error: created.error }, { status: 500 });
  }
  // At this point you can return the key to your user in your UI.
  // In this example we'll just redirect to another page.

  return NextResponse.redirect(new URL(`/keys/verify?key=${created.result.key}`, url));
}
