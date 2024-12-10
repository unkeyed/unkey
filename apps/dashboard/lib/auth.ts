import { auth } from "@/lib/auth/index";
import { redirect } from "next/navigation";

/**
 * Return the tenant id or a 404 not found page.
 *
 * The auth check should already be done at a higher level, and we're just returning 404 to make typescript happy.
 */
export async function getTenantId(): Promise<string> {
  const { userId, orgId } = await auth.getCurrentUser();
  
  if (!userId) {
    console.log("get tenant id: no userId")
    return redirect("/auth/sign-in");
  }
  
  if (!orgId) {
    console.log("get tenant id: no orgrId")
    return redirect("/new");
  }
  
  return orgId;
}
