import { auth } from "./server";

/** Removes team access while preserving the original workspace creator. */
export async function deactivateNonCreatorMemberships(orgId: string): Promise<void> {
  const [memberships, invitations] = await Promise.all([
    auth.getOrganizationMemberList(orgId).catch((error: unknown) => {
      console.error("Failed to list memberships for deactivation:", { orgId, error });
      return null;
    }),
    auth.getInvitationList(orgId).catch((error: unknown) => {
      console.error("Failed to list invitations for revocation:", { orgId, error });
      return null;
    }),
  ]);

  const sorted = [...(memberships?.data ?? [])].sort((a, b) =>
    a.createdAt.localeCompare(b.createdAt),
  );
  const [, ...nonCreators] = sorted;
  const pendingInvitations = (invitations?.data ?? []).filter(
    (invitation) => invitation.state === "pending",
  );

  await Promise.all([
    ...nonCreators.map(async (member) => {
      try {
        await auth.deactivateMembership(member.id, orgId);
      } catch (error) {
        console.error("Failed to deactivate membership:", {
          orgId,
          membershipId: member.id,
          error,
        });
      }
    }),
    ...pendingInvitations.map(async (invitation) => {
      try {
        await auth.revokeOrgInvitation(invitation.id, orgId);
      } catch (error) {
        console.error("Failed to revoke invitation:", {
          orgId,
          invitationId: invitation.id,
          error,
        });
      }
    }),
  ]);
}
