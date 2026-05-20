"use client";

import { IdentityTableActions } from "@/app/(app)/[workspaceSlug]/identities/_components/table/identity-table-actions";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { Gear } from "@unkey/icons";
import { useRouter } from "next/navigation";

export const IdentitySettingsDialog = ({ identityId }: { identityId: string }) => {
  const { data: identity } = trpc.identity.getById.useQuery({ identityId });
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const trpcUtils = trpc.useUtils();

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
        onDeleted={async () => {
          // Wait for the refetch kicked off by `useDeleteIdentity` to complete
          // before navigating, so the destination list renders without the
          // just-deleted row instead of flickering it in then out.
          await trpcUtils.identity.query.invalidate(undefined, { refetchType: "all" });
          router.push(`/${workspace.slug}/identities`);
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
