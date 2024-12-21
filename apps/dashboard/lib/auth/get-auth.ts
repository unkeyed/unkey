import { Membership, UNKEY_SESSION_COOKIE } from './types';
import { auth } from './server';
import { getCookie } from './cookies';

type GetAuthResult = {
  userId: string | null;
  orgId: string | null;
  orgRole: "admin" | "basic_member" | null;
};

export async function getAuth(req?: Request): Promise<GetAuthResult> {
  try {
    const sessionToken = await getCookie(UNKEY_SESSION_COOKIE);

    if (!sessionToken) {
      return {
        userId: null,
        orgId: null,
        orgRole: null
      };
    }

    // Validate session
    const session = await auth.validateSession(sessionToken);
  
    if (!session) {
      return {
        userId: null,
        orgId: null,
        orgRole: null
      };
    }
  
    // fetch org from memberships
    if (session.orgId) {
      const memberships = await auth.getOrganizationMemberList(session.orgId);
      const userMembership = memberships.data.find((m: Membership) => m.user.id === session.userId);
      
      return {
        userId: session.userId,
        orgId: session.orgId,
        orgRole: userMembership?.role ?? null
      };
    }
  
    return {
      userId: session.userId,
      orgId: session.orgId,
      orgRole: null
    };

    }
  catch (error) {
      console.error('Auth validation error:', error);
      return {
        userId: null,
        orgId: null,
        orgRole: null
      };
  }
}