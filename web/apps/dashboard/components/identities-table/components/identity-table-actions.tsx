"use client";

import { DeleteIdentityDialog } from "@/app/(app)/[workspaceSlug]/identities/_components/dialogs/delete-identity-dialog";
import { EditRatelimitDialog } from "@/app/(app)/[workspaceSlug]/identities/_components/dialogs/edit-ratelimit-dialog";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import type { IdentityResponseSchema } from "@/lib/trpc/routers/identity/query";
import { Clone, Code, Gauge, Trash } from "@unkey/icons";
import { toast } from "@unkey/ui";
import { useMemo } from "react";
import type { z } from "zod";
import { EditMetadataDialog } from "./edit-metadata-dialog";

type Identity = z.infer<typeof IdentityResponseSchema>;

export const IdentityTableActions = ({ identity }: { identity: Identity }) => {
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
        ActionComponent: (props) => <DeleteIdentityDialog {...props} identity={identity} />,
      },
    ],
    [identity],
  );

  return <TableActionPopover items={menuItems} />;
};
