import { switchToOrg } from "@/lib/auth";
import { getAvailableWorkspaces } from "@/lib/auth/available-workspaces";
import { getAuth } from "@/lib/auth/get-auth";
import { sanitizeRedirectPath } from "@/lib/auth/redirect-utils";
import { logOperation } from "@/lib/logging";
import { type NextRequest, NextResponse } from "next/server";

const ORGANIZATION_ID = /^[A-Za-z0-9_-]{3,128}$/;

export async function GET(request: NextRequest): Promise<Response> {
  const organizationIds = request.nextUrl.searchParams.getAll("organization_id");
  if (organizationIds.length !== 1 || !ORGANIZATION_ID.test(organizationIds[0])) {
    return NextResponse.redirect(new URL("/auth/error?reason=session", request.url));
  }

  const returnTo = sanitizeRedirectPath(request.nextUrl.searchParams.get("return_to"));
  try {
    const { userId } = await getAuth(request);
    const workspaces = userId ? await getAvailableWorkspaces(userId, organizationIds[0]) : [];
    if (!workspaces.some((workspace) => workspace.orgId === organizationIds[0])) {
      return NextResponse.redirect(new URL("/auth/error?reason=session", request.url));
    }
  } catch (error) {
    logOperation("warn", "Organization switch rejected", {
      organization_id: organizationIds[0],
      error_type: error instanceof Error ? error.constructor.name : typeof error,
      error_message: error instanceof Error ? error.message : "Unknown error",
    });
    return NextResponse.redirect(new URL("/auth/error?reason=session", request.url));
  }
  await switchToOrg(organizationIds[0]);
  return NextResponse.redirect(new URL(returnTo, request.url));
}
