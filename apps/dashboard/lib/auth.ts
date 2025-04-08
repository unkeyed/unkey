import { auth } from "@/lib/auth/server";
import { redirect } from "next/navigation";

/**
 * Return the org id or a 404 not found page.
 *
 * The auth check should already be done at a higher level, and we're just returning 404 to make typescript happy.
 */
export async function getOrgId(): Promise<string> {
  const user = await auth.getCurrentUser();
  if (!user) {
    redirect("/auth/sign-in");
  }

  const { orgId } = user;
  if (!orgId) {
    redirect("/new");
  }

  return orgId;
}

export async function getIsImpersonator(): Promise<boolean> {
  const user = await auth.getCurrentUser();
  if (!user) {
    return false;
  }
  return user.impersonator !== undefined;
}
