import { auth } from "@/lib/auth/server";
import { redirect } from "next/navigation";
import type { NextRequest } from "next/server";

export const dynamic = "force-dynamic";

export async function GET(_request: NextRequest) {
  const user = await auth.getCurrentUser();
  if (!user) {
    return redirect("/auth/sign-in");
  }
  if (!user.orgId) {
    const newOrgId = await auth.createTenant({
      name: "Personal",
      userId: user.id,
    });

    // switch into the new org/workspace
    await auth.switchOrg(newOrgId);
  }

  return redirect("/new?refresh=true");
}
