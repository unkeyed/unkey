"use client";

import { DeleteDialog } from "@/app/(app)/[workspaceSlug]/ratelimits/[namespaceId]/_components/delete-dialog";
import { IdentifierDialog } from "@/app/(app)/[workspaceSlug]/ratelimits/[namespaceId]/_components/identifier-dialog";
import type { OverrideDetails } from "@/app/(app)/[workspaceSlug]/ratelimits/[namespaceId]/types";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { Loading, toast } from "@unkey/ui";
import {
  IconCloneOutline18,
  IconPenWriting3Outline18,
  IconTrashOutline18,
} from "nucleo-ui-outline-18";
import { Suspense } from "react";

export const OverridesTableAction = ({
  identifier,
  namespaceId,
  overrideDetails,
}: {
  identifier: string;
  namespaceId: string;
  overrideDetails?: OverrideDetails | null;
}) => {
  const getOverridesTableActionItems = (): MenuItem[] => {
    return [
      {
        id: "copy",
        label: "Copy identifier",
        icon: <IconCloneOutline18 className="size-4" />,
        onClick: (e) => {
          e.stopPropagation();
          navigator.clipboard
            .writeText(identifier)
            .then(() => {
              toast.success("Copied to clipboard", {
                description: identifier,
              });
            })
            .catch((error) => {
              console.error("Failed to copy to clipboard:", error);
              toast.error("Failed to copy to clipboard");
            });
        },
      },
      {
        id: "override",
        label: "Override Identifier",
        icon: <IconPenWriting3Outline18 className="size-4 text-orange-11" />,
        className: "text-orange-11 hover:bg-orange-2 focus:bg-orange-3",
        ActionComponent: (props) => (
          <IdentifierDialog
            overrideDetails={overrideDetails}
            namespaceId={namespaceId}
            identifier={identifier}
            isModalOpen={props.isOpen}
            onOpenChange={(open) => !open && props.onClose()}
          />
        ),
        divider: true,
      },
      {
        id: "delete",
        label: "Delete Override",
        icon: <IconTrashOutline18 className="size-4 text-error-11" />,
        className: "text-error-11 hover:bg-error-3 focus:bg-error-3",
        ActionComponent: (props) =>
          overrideDetails?.overrideId ? (
            <DeleteDialog
              isModalOpen={props.isOpen}
              onOpenChange={(open) => !open && props.onClose()}
              overrideId={overrideDetails.overrideId}
              identifier={identifier}
            />
          ) : undefined,
        disabled: !overrideDetails?.overrideId,
      },
    ];
  };

  const menuItems = getOverridesTableActionItems();

  return (
    <Suspense fallback={<Loading type="spinner" />}>
      <TableActionPopover items={menuItems} />
    </Suspense>
  );
};
