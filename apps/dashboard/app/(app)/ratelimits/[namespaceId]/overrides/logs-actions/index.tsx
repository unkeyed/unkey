"use client";

import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { Clone, PenWriting3, Trash } from "@unkey/icons";
import { toast } from "@unkey/ui";
import { DeleteDialog } from "../../_components/delete-dialog";
import { IdentifierDialog } from "../../_components/identifier-dialog";
import type { OverrideDetails } from "../../types";

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
        icon: <Clone size="md-regular" />,
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
        icon: <PenWriting3 size="md-regular" className="text-orange-11" />,
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
        icon: <Trash size="md-regular" className="text-error-11" />,
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

  return <TableActionPopover items={menuItems} />;
};
