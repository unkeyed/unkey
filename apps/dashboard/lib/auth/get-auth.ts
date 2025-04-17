import { getCookie } from "./cookies";
import { auth } from "./server";
import { UNKEY_SESSION_COOKIE } from "./types";

type GetAuthResult = {
  userId: string | null;
  orgId: string | null;
};

export async function getAuth(_req?: Request): Promise<GetAuthResult> {
  try {
    const sessionToken = await getCookie(UNKEY_SESSION_COOKIE);

    if (!sessionToken) {
      return {
        userId: null,
        orgId: null,
      };
    }

    // Validate session
    const validationResult = await auth.validateSession(sessionToken);

    let userId: string | undefined;
    let orgId: string | null | undefined;

    if (!validationResult.isValid) {
      if (validationResult.shouldRefresh) {
        try {
          const refreshedData = await auth.refreshSession(sessionToken);
          if (!refreshedData.session) {
            return { userId: null, orgId: null };
          }
          userId = refreshedData.session.userId;
          orgId = refreshedData.session.orgId;
        } catch (error) {
          console.error(error);
          return { userId: null, orgId: null };
        }
      } else {
        return { userId: null, orgId: null };
      }
    } else {
      userId = validationResult.userId;
      orgId = validationResult.orgId;
    }

    // we should have user data from either validation or refresh
    if (!userId) {
      return { userId: null, orgId: null };
    }

    return {
      userId,
      orgId: orgId ?? null,
    };
  } catch (error) {
    console.error("Auth validation error:", error);
    return {
      userId: null,
      orgId: null,
    };
  }
}
