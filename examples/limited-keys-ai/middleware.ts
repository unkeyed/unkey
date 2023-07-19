import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";
import { env } from "./env.mjs";
import { Unkey } from "@unkey/api";

const unkey = new Unkey({ token: env.UNKEY_TOKEN });
export async function middleware(request: NextRequest) {
  const cookie = request.cookies.get("unkey-limited-key");
  const { searchParams } = new URL(request.url);
  const url = new URL(request.url);

  const isValid = searchParams.get("valid");
  console.log("cookie is", cookie);
  if (!cookie || cookie.value.length === 0) {
    return NextResponse.redirect(`${url.origin}/auth`);
  }
  const key = cookie.value;
  const response = NextResponse.next();

  try {
    const { valid } = await unkey.keys.verify({ key });
    if (!valid && isValid === null) {
      return NextResponse.redirect(`${url.origin}?valid=false`);
    }
  } catch (error) {
    return NextResponse.redirect(`${url.origin}/auth`);
  }

  return response;
}
export const config = {
  matcher: ["/api/completions", "/"],
};
