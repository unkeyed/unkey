import type { NextRequest } from "next/server";
import { updateSession } from "./sessions";
import type { User } from "./types";

type GetAuthResult = {
  userId: string | null;
  orgId: string | null;
  accessToken?: string;
  permissions?: readonly string[];
  role: string | null;
  // Profile embedded in the sealed session cookie, when available
  user?: User | null;
  impersonator?: {
    email: string;
    reason?: string | null;
  };
};

export async function getAuth(req?: NextRequest): Promise<GetAuthResult> {
  try {
    // Use the updateSession function which now handles both with and without request
    const { session } = await updateSession(req);

    if (!session) {
      return {
        userId: null,
        orgId: null,
        role: null,
        user: null,
      };
    }

    return session;
  } catch (_error) {
    return {
      userId: null,
      orgId: null,
      role: null,
      user: null,
    };
  }
}
