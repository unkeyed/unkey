"use client";

import { DeleteIdentityDialog } from "@/app/(app)/[workspaceSlug]/identities/_components/dialogs/delete-identity-dialog";
import { EditRatelimitDialog } from "@/app/(app)/[workspaceSlug]/identities/_components/dialogs/edit-ratelimit-dialog";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import type { IdentityForActions } from "@/lib/trpc/routers/identity/query";
import { Clone, Code, Gauge, Trash } from "@unkey/icons";
import { toast } from "@unkey/ui";
import { type PropsWithChildren, useMemo } from "react";
import { EditMetadataDialog } from "./edit-metadata-dialog";

type Identity = IdentityForActions;

export const IdentityTableActions = ({
  identity,
  children,
  onDeleted,
}: PropsWithChildren<{ identity: Identity; onDeleted?: () => void }>) => {
  const menuItems: MenuItem[] = useMemo(
    () => [
      {
        id: "edit-ratelimit",
        label: "Edit ratelimit...",
        icon: <Gauge iconSize="md-medium" />,
        ActionComponent: (props) => <EditRatelimitDialog {...props} identity={identity} />,
      },
      {
        id: "edit-metadata",
        label: "Edit metadata...",
        icon: <Code iconSize="md-medium" />,
        ActionComponent: (props) => <EditMetadataDialog {...props} identity={identity} />,
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
        ActionComponent: (props) => (
          <DeleteIdentityDialog {...props} identity={identity} onDeleted={onDeleted} />
        ),
      },
    ],
    [identity, onDeleted],
  );

  // `children`, when provided, becomes the popover trigger (e.g. the "Settings"
  // button on the identity detail page). When omitted, TableActionPopover
  // falls back to its default `...` trigger used in the identities table row.
  return <TableActionPopover items={menuItems}>{children}</TableActionPopover>;
};
