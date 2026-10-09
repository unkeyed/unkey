"use client";

import { EnvironmentLabel } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/environment-label";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import {
  IconDotsOutline18,
  IconGripDotsVerticalOutline18,
  IconPenWriting3Outline18,
  IconTrashOutline18,
} from "@unkey/icons";
import { match } from "@unkey/match";
import { Button, ConfirmPopover, InfoTooltip, ResourceListItem, ResourceListRow } from "@unkey/ui";
import { cn } from "cn";
import { useRef, useState } from "react";
import type { PolicySwitches } from "../../hooks/policy-switches";
import { POLICY_KINDS } from "../../policy-kinds";
import type { MergedPolicy, PolicyEnvs } from "./merge";
import { type PolicyEnvBadge, policyRowState } from "./row-state";

export const POLICY_COLUMNS =
  "grid grid-cols-[56px_minmax(0,1.4fr)_minmax(0,1fr)_minmax(0,1fr)_32px] items-center gap-4";

export function PoliciesListHeader() {
  return (
    <div
      className={cn(
        POLICY_COLUMNS,
        "border-b bg-table-header px-4 py-[7px] text-xs font-medium text-gray-12",
      )}
    >
      <span>Order</span>
      <span>Name</span>
      <span>Type</span>
      <span>Environments</span>
      <span />
    </div>
  );
}

type PolicyRowProps = {
  policy: MergedPolicy;
  switches: PolicySwitches;
  index: number;
  envs: PolicyEnvs;
  isDragOver: boolean;
  onDelete: (key: string) => void;
  onEdit: (key: string) => void;
  onDragStart: (index: number) => void;
  onDragOver: (index: number) => void;
  onDrop: (index: number) => void;
  onDragEnd: () => void;
};

export function PolicyRow({
  policy,
  switches,
  index,
  envs,
  isDragOver,
  onDelete,
  onEdit,
  onDragStart,
  onDragOver,
  onDrop,
  onDragEnd,
}: PolicyRowProps) {
  const fromHandle = useRef(false);
  const menuButtonRef = useRef<HTMLButtonElement>(null);
  const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false);
  const kind = POLICY_KINDS[policy.type];
  const state = policyRowState(switches, envs);
  const dim = state.dimmed && "opacity-55";

  const edit = () => onEdit(policy.key);

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
      icon: <IconTrashOutline18 className="size-3.5" />,
      onClick: (e) => {
        e.stopPropagation();
        setIsDeleteConfirmOpen(true);
      },
    },
  ];

  return (
    <ResourceListItem
      draggable
      onDragStart={(e) => {
        if (!fromHandle.current) {
          e.preventDefault();
          return;
        }
        e.dataTransfer.effectAllowed = "move";
        setRowDragImage(e);
        onDragStart(index);
      }}
      onDragOver={(e) => {
        e.preventDefault();
        e.dataTransfer.dropEffect = "move";
        onDragOver(index);
      }}
      onDrop={(e) => {
        e.preventDefault();
        onDrop(index);
      }}
      onDragEnd={() => {
        fromHandle.current = false;
        onDragEnd();
      }}
      // If text under the pointer is selected, the browser drags the selection
      // and not this row. It then shows a large page area.
      className={cn("select-none", isDragOver && "bg-grayA-3")}
    >
      <ResourceListRow
        label={`Edit ${policy.name}`}
        onActivate={edit}
        className={cn(POLICY_COLUMNS, "group text-left")}
      >
        <span className={cn("flex items-center gap-2", dim)}>
          <span className="flex size-6 shrink-0 items-center justify-center rounded-md border bg-raised text-xs font-medium tabular-nums text-gray-12">
            {index + 1}
          </span>
          <button
            type="button"
            aria-label={`Drag to reorder ${policy.name}`}
            className="relative flex h-6 w-4 cursor-grab touch-none items-center justify-center text-gray-9 hover:text-gray-11 active:cursor-grabbing disabled:cursor-not-allowed disabled:opacity-50"
            onMouseDown={() => {
              fromHandle.current = true;
            }}
          >
            <IconGripDotsVerticalOutline18 className="size-3.5" />
          </button>
        </span>

        <span className={cn("truncate text-sm font-medium text-gray-12", dim)}>{policy.name}</span>

        <span className={cn("flex min-w-0 items-center gap-1.5 text-sm text-gray-11", dim)}>
          <kind.Icon className="size-3.5 shrink-0" />
          <span className="truncate">{kind.label}</span>
        </span>

        <span className="flex items-center gap-4">
          {state.environments.map((badge) => (
            <EnvBadge key={badge.env} badge={badge} />
          ))}
        </span>

        <span className="relative flex justify-end">
          <TableActionPopover items={menuItems}>
            <Button
              ref={menuButtonRef}
              variant="outline"
              aria-label={`Actions for ${policy.name}`}
              className="size-5 rounded-sm border-transparent [&_svg]:size-3 group-hover:border-strong"
            >
              <IconDotsOutline18 className="text-gray-11 group-hover:text-gray-12" />
            </Button>
          </TableActionPopover>
        </span>
      </ResourceListRow>
      <ConfirmPopover
        isOpen={isDeleteConfirmOpen}
        onOpenChange={setIsDeleteConfirmOpen}
        onConfirm={() => onDelete(policy.key)}
        triggerRef={menuButtonRef}
        title="Confirm deletion"
        description={`Permanently delete "${policy.name}" from every environment?`}
        confirmButtonText="Delete policy"
        cancelButtonText="Cancel"
        variant="danger"
        popoverProps={{ align: "end" }}
      />
    </ResourceListItem>
  );
}

function EnvBadge({ badge }: { badge: PolicyEnvBadge }) {
  const environment = { slug: badge.slug, kind: badge.env };
  return match(badge)
    .with({ enabled: true }, () => (
      <span className="flex">
        <EnvironmentLabel environment={environment} />
      </span>
    ))
    .with({ enabled: false }, () => (
      <InfoTooltip
        asChild
        position={{ side: "top" }}
        content="Requests skip this policy here. Open the policy to turn it on."
      >
        <span className="relative flex items-center gap-1.5">
          <EnvironmentLabel environment={environment} className="text-gray-9" />
          <span className="rounded-sm border border-grayA-4 px-1 text-2xs leading-4 text-gray-10">
            Off
          </span>
        </span>
      </InfoTooltip>
    ))
    .exhaustive();
}

/**
 * Sets the drag image to a detached copy of the row.
 *
 * The browser selects the drag image. For this list it captures a large page
 * area and not the row. The live row does not work as the drag image, because
 * React replaces that node while the drag runs. A copy on `document.body` is
 * outside the render tree, so nothing replaces it.
 */
function setRowDragImage(e: React.DragEvent<HTMLLIElement>) {
  const source = e.currentTarget;
  const { width, height } = source.getBoundingClientRect();
  const clone = source.cloneNode(true);
  if (!(clone instanceof HTMLElement)) {
    return;
  }

  clone.classList.add("rounded-lg", "bg-raised", "shadow-floating", "list-none");

  clone.style.position = "fixed";
  // Keep the copy off-screen but laid out. The browser captures a blank
  // image if the element is not rendered.
  clone.style.top = "-10000px";
  clone.style.left = "-10000px";
  clone.style.width = `${width}px`;
  clone.style.height = `${height}px`;
  clone.style.pointerEvents = "none";
  document.body.appendChild(clone);

  e.dataTransfer.setDragImage(clone, 16, height / 2);
  // The snapshot is taken synchronously, so the clone is disposable.
  requestAnimationFrame(() => clone.remove());
}
