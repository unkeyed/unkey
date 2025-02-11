import { getCookie } from "./cookies";
import { auth } from "./server";
import { type Membership, UNKEY_SESSION_COOKIE } from "./types";

type GetAuthResult = {
  userId: string | null;
  orgId: string | null;
  orgRole: string | null;
};

export async function getAuth(_req?: Request): Promise<GetAuthResult> {
  try {
    const sessionToken = await getCookie(UNKEY_SESSION_COOKIE);

    if (!sessionToken) {
      return {
        userId: null,
        orgId: null,
        orgRole: null,
      };
    }

    // Validate session
    const validationResult = await auth.validateSession(sessionToken);

    let userId: string | undefined;
    let orgId: string | null | undefined;

    if (!validationResult.isValid) {
      if (validationResult.shouldRefresh) {
        try {
          const refreshedData = await auth.refreshSession();
          if (!refreshedData) {
            return { userId: null, orgId: null, orgRole: null };
          }
          userId = refreshedData.userId;
          orgId = refreshedData.orgId;
        } catch (error) {
          console.error(error);
          return { userId: null, orgId: null, orgRole: null };
        }
      } else {
        return { userId: null, orgId: null, orgRole: null };
      }
    } else {
      userId = validationResult.userId;
      orgId = validationResult.orgId;
    }

    // we should have user data from either validation or refresh
    if (!userId) {
      return { userId: null, orgId: null, orgRole: null };
    }

    // fetch org from memberships if we have an org
    if (orgId) {
      const memberships = await auth.getOrganizationMemberList(orgId);
      const userMembership = memberships.data.find((m: Membership) => m.user.id === userId);

      return {
        userId,
        orgId,
        orgRole: userMembership?.role ?? null,
      };
    }

    return {
      userId,
      orgId: orgId ?? null,
      orgRole: null,
    };
  } catch (error) {
    console.error("Auth validation error:", error);
    return {
      userId: null,
      orgId: null,
      orgRole: null,
    };
  }
}
