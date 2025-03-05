import { auth } from "@/lib/auth/server";
import { NextRequest, NextResponse } from "next/server";
import { switchOrg } from "@/lib/auth/actions";
import { redirect } from "next/navigation";

export async function GET(_request: NextRequest) {
  const user = await auth.getCurrentUser();

  // If no user, redirect to sign-in
  if (!user) {
    return redirect("/auth/sign-in");
  }
  
  try {
    // Create a new tenant and get the orgId
    await auth.createTenant({
      name: "Personal",
      userId: user.id,
    }).then(async(newOrgId) => {
      return await switchOrg(newOrgId)
      .then(() => {
        return redirect("/new?refresh=true");
      })
      .catch(error => {
        console.error("Error switching organization:", error);
        return redirect("/error"); // fake page, just needed so /new doesn't continously hit this page and make a bajillions orgs
      });
    })
    
    // The server action handles all the session refreshing, so we can just use that to make sure we're being consistent
    // chaining the promises ensures that we're not returning before the org context updates
    
  } catch (error) {
    console.error("Error creating tenant:", error);
    return redirect("/error"); // fake page, just needed so /new doesn't continously hit this page and make a bajillions orgs
  }
}