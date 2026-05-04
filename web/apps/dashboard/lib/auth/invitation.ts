import { auth } from "./server";

/**
 * Validates that the supplied email matches the email recorded for an
 * invitation token. Throws a single generic error for any failure path
 * (token lookup failed, token resolved but emails differ) so callers cannot
 * use this helper to enumerate the email registered to a token.
 *
 * This module is intentionally not a Server Action ('use server') so it
 * cannot be invoked directly from the network.
 */
export async function requireEmailMatch(params: {
  email: string;
  invitationToken: string;
}): Promise<void> {
  const { email, invitationToken } = params;
  try {
    const invitation = await auth.getInvitation(invitationToken);
    if (invitation?.email === email) {
      return;
    }
  } catch {
    // Fall through to the generic error below.
  }
  throw new Error("Invalid invitation");
}
