import { auth } from "@clerk/nextjs";
import { notFound, redirect } from "next/navigation";

/**
 * Return the tenant id or a 404 not found page.
 *
 * The auth check should already be done at a higher level, and we're just returning 404 to make typescript happy.
 */
export function getTenantId(): string {
  const { userId, orgId } = auth();
  return orgId ?? userId ?? redirect("/auth/sign-in");
}
