import { switchToOrg } from "@/lib/auth";
import { sanitizeRedirectPath } from "@/lib/auth/redirect-utils";
import { type NextRequest, NextResponse } from "next/server";

const ORGANIZATION_ID = /^[A-Za-z0-9_-]{3,128}$/;

export async function GET(request: NextRequest): Promise<Response> {
  const organizationIds = request.nextUrl.searchParams.getAll("organization_id");
  if (organizationIds.length !== 1 || !ORGANIZATION_ID.test(organizationIds[0])) {
    return NextResponse.redirect(new URL("/auth/error?reason=session", request.url));
  }

  const returnTo = sanitizeRedirectPath(request.nextUrl.searchParams.get("return_to"));
  await switchToOrg(organizationIds[0]);
  return NextResponse.redirect(new URL(returnTo, request.url));
}
