"use client";

import { IdentityTableActions } from "@/components/identities-table/components/identity-table-actions";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useIdentity } from "@/lib/identities-query";
import { routes } from "@/lib/navigation/routes";
import { useRouter } from "next/navigation";
import { IconGearOutline18 } from "nucleo-ui-outline-18";

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
        <IconGearOutline18 />
        Retry Settings
      </NavbarActionButton>
    );
  }

  if (!identity) {
    return (
      <NavbarActionButton variant="outline" disabled>
        <IconGearOutline18 />
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
          <IconGearOutline18 />
          Settings
        </NavbarActionButton>
      </IdentityTableActions>
    </div>
  );
};
