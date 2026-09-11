import { auth } from "./server";

/** Removes team access while preserving the original workspace creator. */
export async function deactivateNonCreatorMemberships(orgId: string): Promise<void> {
  const memberships = await auth.getOrganizationMemberList(orgId).catch((error: unknown) => {
    console.error("Failed to list memberships for deactivation:", { orgId, error });
    return null;
  });

  const sorted = [...(memberships?.data ?? [])].sort((a, b) =>
    a.createdAt.localeCompare(b.createdAt),
  );
  const [, ...nonCreators] = sorted;

  await Promise.all(
    nonCreators.map(async (member) => {
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
  );
}
