import { clerkClient } from "@clerk/nextjs";
import { chunkArray } from "./utils";

type ClerkUserData = { id: string; name: string; email: string; tenantId: string };

export async function getClerkOrganizationUsers(tenantId: string): Promise<ClerkUserData[]> {
  const userIds: string[] = [];
  if (tenantId.startsWith("org_")) {
    const members = await clerkClient.organizations.getOrganizationMembershipList({
      organizationId: tenantId,
    });
    for (const m of members) {
      userIds.push(m.publicUserData!.userId);
    }
  } else {
    userIds.push(tenantId);
  }

  return await fetchUserDetails(userIds.map((id) => ({ id, tenantId })));
}

export async function getClerkOrganizationsAdmins(tenantIds: string[]): Promise<ClerkUserData[]> {
  const userIds: { id: string; tenantId: string }[] = [];
  const organizationIds: string[] = [];

  tenantIds.forEach((tenantId) => {
    if (tenantId.startsWith("org_")) {
      organizationIds.push(tenantId);
    } else {
      userIds.push({ id: tenantId, tenantId });
    }
  });

  const adminUserIdsFromOrgs = await fetchAdminUserIdsFromOrganizations(organizationIds);
  userIds.push(...adminUserIdsFromOrgs);

  return await fetchUserDetails(userIds);
}

async function fetchUserDetails(
  userTenantPairs: { id: string; tenantId: string }[],
): Promise<ClerkUserData[]> {
  const userBatches = chunkArray(userTenantPairs, 90);
  const users: ClerkUserData[] = [];

  for await (const batch of userBatches) {
    const details = await Promise.all(
      batch.map(async (userTenantPair) => {
        const user = await clerkClient.users.getUser(userTenantPair.id);

        const email = user.emailAddresses.at(0)?.emailAddress;

        if (!email) {
          throw new Error(`user ${user.id} does not have an email`);
        }

        return {
          id: user.id,
          name: user.firstName ?? user.username ?? "there",
          email,
          tenantId: userTenantPair.tenantId,
        };
      }),
    );

    users.push(...details);
  }

  return users;
}

async function fetchAdminUserIdsFromOrganizations(
  organizationIds: string[],
): Promise<{ id: string; tenantId: string }[]> {
  /// @dev Clerk BE rate limit is 100 requests per 10 seconds.
  const orgBatches = chunkArray(organizationIds, 90);
  const adminUserIds: { id: string; tenantId: string }[] = [];

  for await (const batch of orgBatches) {
    const batchAdminUserIds = await Promise.all(
      batch.map(async (orgId) => {
        const members = await clerkClient.organizations.getOrganizationMembershipList({
          organizationId: orgId,
        });
        /// @dev This only takes the first admin of the organization.
        const adminUserId = members.find((m) => m.role === "admin")?.publicUserData?.userId;

        if (!adminUserId) {
          return null;
        }

        return { id: adminUserId, tenantId: orgId };
      }),
    );

    adminUserIds.push(...(batchAdminUserIds.filter(Boolean) as { id: string; tenantId: string }[]));
  }

  return adminUserIds;
}
