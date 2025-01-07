"use server";

import { clerkClient } from "@clerk/nextjs/server";
import { Filter } from "./filter";

export const UserFilter: React.FC<{ tenantId: string }> = async ({ tenantId }) => {
  if (tenantId.startsWith("user_")) {
    return null;
  }

  const members = await clerkClient.organizations.getOrganizationMembershipList({
    organizationId: tenantId,
  });

  return (
    <Filter
      param="users"
      title="Users"
      options={members
        .filter((m) => Boolean(m.publicUserData))
        .map((m) => ({
          label:
            m.publicUserData!.firstName && m.publicUserData!.lastName
              ? `${m.publicUserData!.firstName} ${m.publicUserData!.lastName}`
              : m.publicUserData!.identifier,
          value: m.publicUserData!.userId,
        }))}
    />
  );
};
