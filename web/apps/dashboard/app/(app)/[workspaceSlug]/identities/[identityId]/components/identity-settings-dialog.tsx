"use client";

import { IdentityTableActions } from "@/components/identities-table/components/identity-table-actions";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useIdentity } from "@/lib/identities-query";
import { routes } from "@/lib/navigation/routes";
import { Gear } from "@unkey/icons";
import { useRouter } from "next/navigation";

export const IdentitySettingsDialog = ({ identityId }: { identityId: string }) => {
  const { data: identity, isError, refetch } = useIdentity(identityId);
  const router = useRouter();
  const workspace = useWorkspaceNavigation();

  if (isError && !identity) {
    return (
      <NavbarActionButton
        variant="outline"
        onClick={() => {
          refetch().catch((error: unknown) => {
            console.error("Failed to retry identity query", error);
          });
        }}
      >
        <Gear />
        Retry Settings
      </NavbarActionButton>
    );
  }

  if (!identity) {
    return (
      <NavbarActionButton variant="outline" disabled>
        <Gear />
        Settings
      </NavbarActionButton>
    );
  }

  return (
    <div>
      <IdentityTableActions
        identity={identity}
        onDeleted={() => {
          router.push(routes.identities.list({ workspaceSlug: workspace.slug }));
        }}
      >
        <NavbarActionButton variant="outline">
          <Gear />
          Settings
        </NavbarActionButton>
      </IdentityTableActions>
    </div>
  );
};
