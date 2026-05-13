"use client";

import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import type { IdentityForActions } from "@/lib/trpc/routers/identity/query";
import { Clone, Code, Gauge, Trash } from "@unkey/icons";
import { toast } from "@unkey/ui";
import { type PropsWithChildren, useMemo, useState } from "react";
import { DeleteIdentityDialog } from "../dialogs/delete-identity-dialog";
import { EditRatelimitDialog } from "../dialogs/edit-ratelimit-dialog";
import { EditMetadataDialog } from "./edit-metadata-dialog";

export const IdentityTableActions = ({
  identity,
  children,
}: PropsWithChildren<{ identity: IdentityForActions }>) => {
  const [isEditMetadataOpen, setIsEditMetadataOpen] = useState(false);
  const [isEditRatelimitOpen, setIsEditRatelimitOpen] = useState(false);
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);

  const menuItems: MenuItem[] = useMemo(
    () => [
      {
        id: "edit-ratelimit",
        label: "Edit ratelimit...",
        icon: <Gauge iconSize="md-medium" />,
        onClick: () => {
          setIsEditRatelimitOpen(true);
        },
      },
      {
        id: "edit-metadata",
        label: "Edit metadata...",
        icon: <Code iconSize="md-medium" />,
        onClick: () => {
          setIsEditMetadataOpen(true);
        },
        divider: true,
      },
      {
        id: "copy-identity-id",
        label: "Copy identity ID",
        icon: <Clone iconSize="md-medium" />,
        onClick: () => {
          navigator.clipboard
            .writeText(identity.id)
            .then(() => {
              toast.success("Identity ID copied to clipboard");
            })
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
        onClick: () => {
          navigator.clipboard
            .writeText(identity.externalId)
            .then(() => {
              toast.success("External ID copied to clipboard");
            })
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
        onClick: () => {
          setIsDeleteDialogOpen(true);
        },
        variant: "danger" as const,
      },
    ],
    [identity.id, identity.externalId],
  );

  return (
    <>
      <TableActionPopover items={menuItems}>{children}</TableActionPopover>
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
      />
    </>
  );
};
