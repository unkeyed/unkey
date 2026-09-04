import { env } from "@/lib/env";
import type { NextRequest } from "next/server";

export const dynamic = "force-dynamic";

export async function GET(request: NextRequest): Promise<Response> {
  if (env().AUTH_PROVIDER === "local") {
    const { handleLocalJoin } = await import("./local-join");
    return handleLocalJoin(request);
  }

  return new Response(null, { status: 404 });
}
