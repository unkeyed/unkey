"use client";

import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import {
  IconCubeOutline18,
  IconDotsOutline18,
  IconPenWriting3Outline18,
  IconTrashOutline18,
  IconTriangleWarningOutline12,
} from "@unkey/icons";
import { Button, CopyButton, InfoTooltip, ResourceListItem, ResourceListRow } from "@unkey/ui";
import { cn } from "cn";
import { useState } from "react";
import type { ConnectionRowModel } from "../../hooks/use-connections-data";
import { connectionHostVariable } from "../../model/name";
import type { ConnectionTarget, ListedConnection } from "../../model/target";
import { DeleteConnectionDialog } from "../delete-connection-dialog";

export function ConnectionsListHeader() {
  return (
    <div className="grid grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)_minmax(0,1.2fr)_32px] items-center gap-4 border-b bg-table-header px-4 py-[7px] text-xs font-medium text-gray-12">
      <span>App</span>
      <span>Variable</span>
      <span>Target</span>
      <span />
    </div>
  );
}

export function ConnectionRow({
  row: { connection, target },
  onEdit,
  onRemove,
  removing,
}: {
  row: ConnectionRowModel;
  onEdit: (connection: ListedConnection) => void;
  onRemove: (connection: ListedConnection) => void;
  removing: boolean;
}) {
  const [isRemoveConfirmOpen, setIsRemoveConfirmOpen] = useState(false);
  const variable = connectionHostVariable(connection.name);
  const edit = () => onEdit(connection);

  const menuItems: MenuItem[] = [
    {
      id: "edit",
      label: "Edit",
      icon: <IconPenWriting3Outline18 className="size-3.5" />,
      divider: true,
      onClick: (e) => {
        e.stopPropagation();
        edit();
      },
    },
    {
      id: "delete",
      label: "Delete",
      disabled: removing,
      icon: <IconTrashOutline18 className="size-3.5" />,
      onClick: (e) => {
        e.stopPropagation();
        setIsRemoveConfirmOpen(true);
      },
    },
  ];

  return (
    <ResourceListItem>
      <ResourceListRow
        label={`Edit ${connection.targetAppName} connection`}
        onActivate={edit}
        className="group grid grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)_minmax(0,1.2fr)_32px] items-center gap-4 text-left"
      >
        <span className="flex min-w-0 items-center gap-2">
          <IconCubeOutline18 className="size-3.5 shrink-0 text-gray-11" />
          <span className="truncate text-sm font-medium text-gray-12">
            {connection.targetAppName}
          </span>
        </span>
        <span className="flex min-w-0 items-center gap-1">
          <code className="truncate font-mono text-xs text-gray-11">{variable}</code>
          <CopyButton
            value={variable}
            variant="ghost"
            className="relative size-5 shrink-0 [&_svg]:size-3"
          />
        </span>
        <span className="flex min-w-0">
          <TargetBadge target={target} />
        </span>
        <span className="relative flex justify-end">
          <TableActionPopover items={menuItems}>
            <Button
              variant="outline"
              aria-label={`Actions for ${connection.targetAppName} connection`}
              className="size-5 rounded-sm border-transparent [&_svg]:size-3 group-hover:border-strong"
            >
              <IconDotsOutline18 className="text-gray-11 group-hover:text-gray-12" />
            </Button>
          </TableActionPopover>
        </span>
      </ResourceListRow>
      <DeleteConnectionDialog
        targetAppName={connection.targetAppName}
        open={isRemoveConfirmOpen}
        onOpenChange={setIsRemoveConfirmOpen}
        onConfirm={() => onRemove(connection)}
      />
    </ResourceListItem>
  );
}

function TargetBadge({ target }: { target: ConnectionTarget }) {
  return (
    <InfoTooltip
      asChild
      position={{ side: "top" }}
      content={<span className="flex max-w-64">{target.text}</span>}
    >
      <span
        className={cn(
          "relative flex min-w-0 items-center gap-1 text-xs",
          target.unavailable ? "rounded-sm bg-warningA-3 px-1 text-warningA-11" : "text-gray-11",
        )}
      >
        {target.unavailable ? <IconTriangleWarningOutline12 className="size-3 shrink-0" /> : null}
        <span className="truncate">{target.label}</span>
      </span>
    </InfoTooltip>
  );
}
