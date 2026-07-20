"use client";

import { DeleteIdentityDialog } from "@/app/(app)/[workspaceSlug]/identities/_components/dialogs/delete-identity-dialog";
import { EditRatelimitDialog } from "@/app/(app)/[workspaceSlug]/identities/_components/dialogs/edit-ratelimit-dialog";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import type { Identity } from "@unkey/api/models/components";
import { Clone, Code, Gauge, Trash } from "@unkey/icons";
import { toast } from "@unkey/ui";
import { type PropsWithChildren, createContext, useContext, useMemo } from "react";
import { EditMetadataDialog } from "./edit-metadata-dialog";

type IdentityActionsContextValue = {
  identity: Identity;
  onDeleted?: () => void;
};

const IdentityActionsContext = createContext<IdentityActionsContextValue | null>(null);

function useIdentityActionsContext(): IdentityActionsContextValue {
  const context = useContext(IdentityActionsContext);
  if (!context) {
    throw new Error("Identity action dialogs must be rendered within IdentityTableActions");
  }
  return context;
}

const EditRatelimitAction: NonNullable<MenuItem["ActionComponent"]> = (props) => {
  const { identity } = useIdentityActionsContext();
  return <EditRatelimitDialog {...props} identity={identity} />;
};

const EditMetadataAction: NonNullable<MenuItem["ActionComponent"]> = (props) => {
  const { identity } = useIdentityActionsContext();
  return <EditMetadataDialog {...props} identity={identity} />;
};

const DeleteIdentityAction: NonNullable<MenuItem["ActionComponent"]> = (props) => {
  const { identity, onDeleted } = useIdentityActionsContext();
  return <DeleteIdentityDialog {...props} identity={identity} onDeleted={onDeleted} />;
};

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
        ActionComponent: EditRatelimitAction,
      },
      {
        id: "edit-metadata",
        label: "Edit metadata...",
        icon: <Code iconSize="md-medium" />,
        ActionComponent: EditMetadataAction,
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
        ActionComponent: DeleteIdentityAction,
      },
    ],
    [identity.externalId, identity.id],
  );

  // `children`, when provided, becomes the popover trigger (e.g. the "Settings"
  // button on the identity detail page). When omitted, TableActionPopover
  // falls back to its default `...` trigger used in the identities list row.
  return (
    <IdentityActionsContext value={{ identity, onDeleted }}>
      <TableActionPopover items={menuItems}>{children}</TableActionPopover>
    </IdentityActionsContext>
  );
};
