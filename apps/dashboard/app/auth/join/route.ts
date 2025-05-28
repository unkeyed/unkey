import { getCurrentUser } from "@/lib/auth";
import { auth } from "@/lib/auth/server";
import { type NextRequest, NextResponse } from "next/server";
import { switchOrg } from "../actions";

export async function GET(request: NextRequest) {
  const dashboardUrl = new URL("/apis", request.url);
  const signInUrl = new URL("/auth/sign-in", request.url);
  const signUpUrl = new URL("/auth/sign-up", request.url);

  const searchParams = request.nextUrl.searchParams;
  const invitationToken = searchParams.get("invitation_token");
  if (!invitationToken) {
    return NextResponse.redirect(dashboardUrl); // middleware will pickup if they are not authenticated and redirect to login
  }

  const user = await getCurrentUser();
  // exchange token for invitation
  const invitation = await auth.getInvitation(invitationToken);
  if (!invitation) {
    console.error(`No invitation found for ${invitationToken}`);
    return NextResponse.redirect(dashboardUrl);
  }

  const { email: invitationEmail, state, organizationId, id: invitationId } = invitation;

  if (state !== "pending") {
    // TODO: better handle accepted/revoked/expired invitations
    console.error(`Unable to accept invitation due to state: ${state}`);
    return NextResponse.redirect(dashboardUrl);
  }

  // if they are authenticated
  if (user) {
    if (user.email !== invitationEmail) {
      console.error("User email does not match invitation");
    } else {
      auth.acceptInvitation(invitationId).then(() => {
        return switchOrg(organizationId!);
      });
    }

    return NextResponse.redirect(dashboardUrl);
  }

  // if they are not authenticated

  const existingUser = await auth.findUser(invitationEmail);

  if (existingUser) {
    signInUrl.searchParams.set("invitation_token", invitationToken);
    signInUrl.searchParams.set("email", invitationEmail);

    return NextResponse.redirect(signInUrl);
  }
  signUpUrl.searchParams.set("invitation_token", invitationToken);
  signUpUrl.searchParams.set("email", invitationEmail);

  return NextResponse.redirect(signUpUrl);
}
