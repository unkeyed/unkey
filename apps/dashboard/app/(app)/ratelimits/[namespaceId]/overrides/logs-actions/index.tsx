"use client";

import { Clone, PenWriting3, Trash } from "@unkey/icons";
import { toast } from "@unkey/ui";
import { useState } from "react";
import { DeleteDialog } from "../../_components/delete-dialog";
import { IdentifierDialog } from "../../_components/identifier-dialog";
import { type MenuItem, TableActionPopover } from "../../_components/table-action-popover";
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
  const [isIdentifierModalOpen, setIsIdentifierModalOpen] = useState(false);
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);

  const items: MenuItem[] = [
    {
      id: "copy",
      label: "Copy identifier",
      icon: <Clone size="md-regular" />,
      onClick: (e) => {
        e.stopPropagation();
        navigator.clipboard.writeText(identifier);
        toast.success("Copied to clipboard", {
          description: identifier,
        });
      },
    },
    {
      id: "override",
      label: "Override Identifier",
      icon: <PenWriting3 size="md-regular" />,
      className: "text-orange-11 hover:bg-orange-2 focus:bg-orange-3",
      onClick: (e) => {
        e.stopPropagation();
        setIsIdentifierModalOpen(true);
      },
    },
    {
      id: "delete",
      label: "Delete Override",
      icon: <Trash size="md-regular" />,
      className: "text-error-11 hover:bg-error-3 focus:bg-error-3",
      onClick: (e) => {
        e.stopPropagation();
        setIsDeleteModalOpen(true);
      },
    },
  ];

  return (
    <>
      <TableActionPopover items={items} />
      <IdentifierDialog
        overrideDetails={overrideDetails}
        namespaceId={namespaceId}
        identifier={identifier}
        isModalOpen={isIdentifierModalOpen}
        onOpenChange={setIsIdentifierModalOpen}
      />

      {overrideDetails?.overrideId && (
        <DeleteDialog
          isModalOpen={isDeleteModalOpen}
          onOpenChange={setIsDeleteModalOpen}
          overrideId={overrideDetails.overrideId}
          identifier={identifier}
        />
      )}
    </>
  );
};
