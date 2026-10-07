import { logOperation } from "@/lib/logging";
import { auth } from "./server";

/**
 * Deactivating in lockstep keeps a large organization from bursting past the
 * provider's rate limit, where the rejected calls would leave members with
 * access that billing already revoked. Kept well above the batch size the
 * provider can absorb: a Stripe redelivery cannot resume this work, because the
 * plan change has already committed by the time it runs, so a timeout here
 * strands members permanently.
 */
const CONCURRENCY = 20;

/** Removes team access while preserving the original workspace creator. */
export async function deactivateNonCreatorMemberships(orgId: string): Promise<void> {
  const memberships = await auth.getOrganizationMemberList(orgId).catch((error: unknown) => {
    // Billing webhooks must not retry forever, so the failure is reported
    // rather than thrown. Nobody loses access on this path.
    console.error("Failed to list memberships for deactivation:", { orgId, error });
    logOperation("error", "Membership deactivation could not list members", {
      auth_event: "membership_deactivation",
      auth_outcome: "failure",
      org_id: orgId,
    });
    return null;
  });

  const sorted = [...(memberships?.data ?? [])].sort((a, b) =>
    a.createdAt.localeCompare(b.createdAt),
  );
  const [, ...nonCreators] = sorted;

  let failures = 0;
  for (let start = 0; start < nonCreators.length; start += CONCURRENCY) {
    const batch = nonCreators.slice(start, start + CONCURRENCY);
    const before = failures;
    await Promise.all(
      batch.map(async (member) => {
        try {
          await auth.deactivateMembership(member.id, orgId);
        } catch (error) {
          failures++;
          console.error("Failed to deactivate membership:", {
            orgId,
            membershipId: member.id,
            error,
          });
        }
      }),
    );

    // Reported per batch, not once at the end: this runs inside a webhook whose
    // budget can expire mid-loop, and a summary that never executes tells the
    // operator nothing about how far revocation got.
    if (failures > before) {
      logOperation("error", "Membership deactivation left members active", {
        auth_event: "membership_deactivation",
        auth_outcome: "failure",
        org_id: orgId,
        failed_count: failures,
        processed_count: start + batch.length,
        total_count: nonCreators.length,
      });
    }
  }
}
