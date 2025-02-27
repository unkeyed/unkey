import { auth } from "@/lib/auth/server";
import { redirect } from "next/navigation";

/**
 * Return the tenant id or a 404 not found page.
 *
 * The auth check should already be done at a higher level, and we're just returning 404 to make typescript happy.
 */
export async function getTenantId(): Promise<string> {
  const user = await auth.getCurrentUser();
  if (!user) {
    return redirect("/auth/sign-in");
  }

  console.log("getTenantId: ", user.orgId)
  const { orgId } = user;
  if (!orgId) {
    return redirect("/new");
  }

  return orgId;
}
