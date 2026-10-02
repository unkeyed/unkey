import { type NextRequest, NextResponse } from "next/server";

export function handleLocalJoin(request: NextRequest): NextResponse {
  return NextResponse.redirect(new URL("/apis", request.url));
}
