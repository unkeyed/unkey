import { Unkey } from "@unkey/api";
import { NextRequest, NextResponse } from "next/server";

export const dynamic = "force-dynamic";

export async function GET(req: NextRequest) {
  const unkeyToken = process.env.UNKEY_ROOT_KEY;
  if (!unkeyToken) {
    return NextResponse.json({ error: "UNKEY_ROOT_KEY is undefined" }, { status: 500 });
  }
  const unkey = new Unkey({ token: unkeyToken });
  const url = new URL(req.url);
  const apiId = process.env.UNKEY_API_ID;
  if (!apiId) {
    return NextResponse.json({ error: "UNKEY_API_ID is undefined" }, { status: 500 });
  }
  const created = await unkey.keys.create({
    apiId,
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
