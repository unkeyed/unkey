import type { NextRequest } from "next/server";
import { updateSession } from "./sessions";

type GetAuthResult = {
  userId: string | null;
  orgId: string | null;
  role: string | null;
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
      };
    }

    return session;
  } catch (_error) {
    return {
      userId: null,
      orgId: null,
      role: null,
    };
  }
}
