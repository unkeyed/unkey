import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";
import { env } from "./env.mjs";
import { Unkey } from "@unkey/api";
const unkey = new Unkey({ token: env.UNKEY_TOKEN });
export async function middleware(request: NextRequest) {
  const cookie = request.cookies.get("unkey-limited-key");
  console.log(cookie);
  if (!cookie) {
    return NextResponse.rewrite(new URL("/auth", request.url));
  }
  const key = cookie.value;

  const { valid } = await unkey.keys.verify({ key });
  if (!valid) {
    return new NextResponse(
      JSON.stringify({ success: false, message: "Limit exceeded" }),
      { status: 401, headers: { "content-type": "application/json" } }
    );
  }
  const response = NextResponse.next();

  return response;
}
export const config = {
  matcher: ["/api/completions", "/"],
};
