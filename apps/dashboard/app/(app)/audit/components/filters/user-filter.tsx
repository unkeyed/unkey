"use server";

import { useOrganization } from "@/lib/auth/hooks/useOrganization";
import { Filter } from "./filter";

export const UserFilter: React.FC<{ tenantId: string }> = async ({ tenantId }) => {
  if (tenantId.startsWith("user_")) {
    return null;
  }

  const { memberships: members }  = useOrganization();

  return (
    <Filter
      param="users"
      title="Users"
      options={members
        .map((m) => ({
          label:
            m.user.fullName
              ? m.user.fullName
              : m.user.email,
          value: m.user.id,
        }))}
    />
  );
};
