"use client";

import { DeleteIdentityDialog } from "@/app/(app)/[workspaceSlug]/identities/_components/dialogs/delete-identity-dialog";
import { EditRatelimitDialog } from "@/app/(app)/[workspaceSlug]/identities/_components/dialogs/edit-ratelimit-dialog";
import { EditMetadataDialog } from "@/app/(app)/[workspaceSlug]/identities/_components/table/edit-metadata-dialog";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { Clone, Code, Gauge, Gear, Trash } from "@unkey/icons";
import { toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";

type IdentitySettingsButtonProps = {
  identityId: string;
};

export const IdentitySettingsButton = ({ identityId }: IdentitySettingsButtonProps) => {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const [isEditMetadataOpen, setIsEditMetadataOpen] = useState(false);
  const [isEditRatelimitOpen, setIsEditRatelimitOpen] = useState(false);
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);

  const { data: identity } = trpc.identity.getById.useQuery({ identityId });

  const menuItems: MenuItem[] = useMemo(() => {
    const disabled = !identity;
    return [
      {
        id: "edit-ratelimit",
        label: "Edit ratelimit...",
        icon: <Gauge iconSize="md-medium" />,
        disabled,
        onClick: () => setIsEditRatelimitOpen(true),
      },
      {
        id: "edit-metadata",
        label: "Edit metadata...",
        icon: <Code iconSize="md-medium" />,
        disabled,
        onClick: () => setIsEditMetadataOpen(true),
        divider: true,
      },
      {
        id: "copy-identity-id",
        label: "Copy identity ID",
        icon: <Clone iconSize="md-medium" />,
        onClick: () => {
          navigator.clipboard
            .writeText(identityId)
            .then(() => toast.success("Identity ID copied to clipboard"))
            .catch((error) => {
              console.error("Failed to copy to clipboard:", error);
              toast.error("Failed to copy to clipboard");
            });
        },
      },
      {
        id: "copy-external-id",
        label: "Copy external ID",
        icon: <Clone iconSize="md-medium" />,
        disabled,
        onClick: () => {
          if (!identity) {
            return;
          }
          navigator.clipboard
            .writeText(identity.externalId)
            .then(() => toast.success("External ID copied to clipboard"))
            .catch((error) => {
              console.error("Failed to copy to clipboard:", error);
              toast.error("Failed to copy to clipboard");
            });
        },
        divider: true,
      },
      {
        id: "delete-identity",
        label: "Delete identity",
        icon: <Trash iconSize="md-medium" />,
        disabled,
        onClick: () => setIsDeleteDialogOpen(true),
      },
    ];
  }, [identity, identityId]);

  return (
    <>
      <TableActionPopover items={menuItems}>
        <NavbarActionButton>
          <Gear />
          Settings
        </NavbarActionButton>
      </TableActionPopover>
      {identity ? (
        <>
          <EditRatelimitDialog
            identity={identity}
            open={isEditRatelimitOpen}
            onOpenChange={setIsEditRatelimitOpen}
          />
          <EditMetadataDialog
            identity={identity}
            open={isEditMetadataOpen}
            onOpenChange={setIsEditMetadataOpen}
          />
          <DeleteIdentityDialog
            identity={identity}
            open={isDeleteDialogOpen}
            onOpenChange={setIsDeleteDialogOpen}
            onDeleted={() => router.push(`/${workspace.slug}/identities`)}
          />
        </>
      ) : null}
    </>
  );
};
