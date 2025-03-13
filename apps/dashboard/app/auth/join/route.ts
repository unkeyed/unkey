import { NextRequest, NextResponse } from 'next/server';
import { auth } from '@/lib/auth/server';
import { switchOrg } from '@/lib/auth/actions';

export async function GET(request: NextRequest) {
    const DASHBOARD_URL = new URL('/apis', request.url);
    const SIGN_IN_URL = new URL('/auth/sign-in', request.url);
    const SIGN_UP_URL = new URL('/auth/sign-up', request.url);

    const searchParams = request.nextUrl.searchParams;
    const invitationToken = searchParams.get("invitation_token");
    if (!invitationToken) {
        return NextResponse.redirect(DASHBOARD_URL); // middleware will pickup if they are not authenticated and redirect to login
    }

    console.log("join page");

    const user = await auth.getCurrentUser();
    // exchange token for invitation
    const invitation = await auth.getInvitation(invitationToken);
    if (!invitation) {
        console.error(`No invitation found for ${invitationToken}`);
        return NextResponse.redirect(DASHBOARD_URL);
    }

    const { email: invitationEmail, state, organizationId, id: invitationId } = invitation;

    if (state !== "pending") {
        // TODO: better handle accepted/revoked/expired invitations
        console.error(`Unable to accept invitation due to state: ${state}`);
        return NextResponse.redirect(DASHBOARD_URL);
    }

    // if they are authenticated
    if (user) {
        if (user.email !== invitationEmail) {
            console.error("User email does not match invitation");
        }

        else {
            auth.acceptInvitation(invitationId)
                .then(() => {
                    return switchOrg(organizationId!);
                });
        }

        return NextResponse.redirect(DASHBOARD_URL);
    }
    
    // if they are not authenticated
    else {
        const existingUser = await auth.findUser(invitationEmail);

        if (existingUser) {
            SIGN_IN_URL.searchParams.set('invitation_token', invitationToken);
            SIGN_IN_URL.searchParams.set('email', invitationEmail);

            return NextResponse.redirect(SIGN_IN_URL);
        }

        else {
            SIGN_UP_URL.searchParams.set('invitation_token', invitationToken);
            SIGN_UP_URL.searchParams.set('email', invitationEmail);

            return NextResponse.redirect(SIGN_UP_URL);
        }

    }

};

