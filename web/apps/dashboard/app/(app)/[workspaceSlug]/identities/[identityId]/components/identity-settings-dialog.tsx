"use client";

import { IdentityTableActions } from "@/app/(app)/[workspaceSlug]/identities/_components/table/identity-table-actions";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { trpc } from "@/lib/trpc/client";
import { Gear } from "@unkey/icons";

export const IdentitySettingsDialog = ({ identityId }: { identityId: string }) => {
  const { data: identity } = trpc.identity.getById.useQuery({ identityId });

  if (!identity) {
    return (
      <NavbarActionButton disabled>
        <Gear />
        Settings
      </NavbarActionButton>
    );
  }

  return (
    <IdentityTableActions identity={identity}>
      <NavbarActionButton>
        <Gear />
        Settings
      </NavbarActionButton>
    </IdentityTableActions>
  );
};
