"use client";

import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { Switch } from "@/components/ui/switch";
import type { SentinelPolicy } from "@/lib/collections/deploy/sentinel-policies.schema";
import { cn } from "@/lib/utils";
import { Dots, GripDotsVertical, PenWriting3, Trash } from "@unkey/icons";
import { Button, ConfirmPopover } from "@unkey/ui";
import { useRef, useState } from "react";

type MergedPolicyRow = {
  id: string;
  name: string;
  type: SentinelPolicy["type"];
  envA: SentinelPolicy | null;
  envB: SentinelPolicy | null;
};

type SentinelPolicyRowProps = {
  policy: MergedPolicyRow;
  index: number;
  isLast: boolean;
  isDragOver: boolean;
  envASlug: string;
  envBSlug: string;
  onToggleEnvA: (id: string) => void;
  onToggleEnvB: (id: string) => void;
  onAddToEnvA: (id: string) => void;
  onAddToEnvB: (id: string) => void;
  onDelete: (id: string) => void;
  onEdit: (policy: SentinelPolicy) => void;
  onDragStart: (index: number) => void;
  onDragOver: (index: number) => void;
  onDrop: (index: number) => void;
  onDragEnd: () => void;
};

const POLICY_TYPE_LABELS: Record<SentinelPolicy["type"], string> = {
  keyauth: "Key Auth",
  ratelimit: "Rate Limit",
  firewall: "Firewall",
};

export function SentinelPolicyRow({
  policy,
  index,
  isLast,
  isDragOver,
  envASlug,
  envBSlug,
  onToggleEnvA,
  onToggleEnvB,
  onAddToEnvA,
  onAddToEnvB,
  onDelete,
  onEdit,
  onDragStart,
  onDragOver,
  onDrop,
  onDragEnd,
}: SentinelPolicyRowProps) {
  const fromHandle = useRef(false);
  const deleteButtonRef = useRef<HTMLButtonElement>(null);
  const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false);

  const menuItems: MenuItem[] = [
    {
      id: "edit",
      label: "Edit",
      icon: <PenWriting3 iconSize="md-regular" />,
      divider: true,
      onClick: (e) => {
        e.stopPropagation();
        const target = policy.envA ?? policy.envB;
        if (target) {
          onEdit(target);
        }
      },
    },
    {
      id: "delete",
      label: "Delete",
      icon: <Trash iconSize="md-regular" />,
      onClick: (e) => {
        e.stopPropagation();
        setIsDeleteConfirmOpen(true);
      },
    },
  ];

  const isActiveAnywhere = (policy.envA?.enabled ?? false) || (policy.envB?.enabled ?? false);

  return (
    <div
      draggable
      onDragStart={(e) => {
        if (!fromHandle.current) {
          e.preventDefault();
          return;
        }
        e.dataTransfer.effectAllowed = "move";
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
      className={cn(!isLast && "border-b border-grayA-4", isDragOver && "bg-grayA-3")}
    >
      <div className={cn(!isActiveAnywhere && "opacity-55")}>
        <button
          type="button"
          className="group flex items-center hover:bg-grayA-2 transition-colors cursor-pointer w-full text-left"
          onClick={() => {
            const target = policy.envA ?? policy.envB;
            if (target) {
              onEdit(target);
            }
          }}
        >
          {/* Step number */}
          <div className="w-10 shrink-0 py-5 pl-4 flex items-center">
            <div
              className={cn(
                "size-6 rounded-full border flex items-center justify-center text-[11px] font-medium",
                isActiveAnywhere
                  ? "bg-info-3 border-info-7 text-info-11"
                  : "bg-grayA-2 border-grayA-5 text-gray-10",
              )}
            >
              {index + 1}
            </div>
          </div>

          {/* Drag handle */}
          <button
            type="button"
            className="w-10 shrink-0 flex items-center justify-center py-5 cursor-grab active:cursor-grabbing touch-none"
            onMouseDown={() => {
              fromHandle.current = true;
            }}
            onClick={(e) => e.stopPropagation()}
          >
            <GripDotsVertical iconSize="lg-medium" className="opacity-40 hover:opacity-70" />
          </button>

          {/* Name */}
          <div className="flex-4 min-w-0 py-5 flex items-center pr-5">
            <span
              className={cn(
                "text-[13px] truncate",
                policy.name ? "text-gray-12" : "text-gray-9 italic",
              )}
            >
              {policy.name || "Untitled policy"}
            </span>
          </div>

          {/* Type */}
          <div className="flex-4 min-w-0 py-5 flex items-center pr-3">
            <span className="text-[13px] text-gray-11 truncate">
              {POLICY_TYPE_LABELS[policy.type]}
            </span>
          </div>

          {/* Env badges */}
          <div className="flex-3 min-w-0 py-5 flex items-center gap-3 pr-3">
            <EnvSwitch
              id={policy.id}
              slug={envASlug}
              envPolicy={policy.envA}
              onToggle={onToggleEnvA}
              onAdd={onAddToEnvA}
            />
            <EnvSwitch
              id={policy.id}
              slug={envBSlug}
              envPolicy={policy.envB}
              onToggle={onToggleEnvB}
              onAdd={onAddToEnvB}
            />
          </div>

          {/* Actions */}
          <div className="w-12 shrink-0 py-5 flex items-center justify-end pr-4">
            <TableActionPopover items={menuItems}>
              <Button
                ref={deleteButtonRef}
                variant="outline"
                className="size-5 [&_svg]:size-3 rounded-sm border-transparent group-hover:border-grayA-6"
                onClick={(e) => e.stopPropagation()}
              >
                <Dots className="group-hover:text-gray-12 text-gray-11" iconSize="sm-regular" />
              </Button>
            </TableActionPopover>

            <ConfirmPopover
              isOpen={isDeleteConfirmOpen}
              onOpenChange={setIsDeleteConfirmOpen}
              onConfirm={() => onDelete(policy.id)}
              triggerRef={deleteButtonRef}
              title="Confirm deletion"
              description={`This will permanently delete "${policy.name}". This action cannot be undone.`}
              confirmButtonText="Delete policy"
              cancelButtonText="Cancel"
              variant="danger"
            />
          </div>
        </button>
      </div>
    </div>
  );
}

function EnvSwitch({
  id,
  slug,
  envPolicy,
  onToggle,
  onAdd,
}: {
  id: string;
  slug: string;
  envPolicy: SentinelPolicy | null;
  onToggle: (id: string) => void;
  onAdd: (id: string) => void;
}) {
  if (envPolicy !== null) {
    return (
      <span
        className="flex items-center gap-2 shrink-0"
        onClick={(e) => e.stopPropagation()}
        onKeyDown={(e) => e.stopPropagation()}
      >
        <span className="text-[13px] text-gray-11 capitalize whitespace-nowrap">{slug}</span>
        <Switch
          checked={envPolicy.enabled}
          onCheckedChange={() => onToggle(id)}
          className="h-5 w-10 data-[state=checked]:bg-info-9 data-[state=checked]:ring-2 data-[state=checked]:ring-infoA-5 data-[state=unchecked]:bg-grayA-6 data-[state=unchecked]:ring-2 data-[state=unchecked]:ring-grayA-5"
          thumbClassName="h-4 w-4 data-[state=checked]:translate-x-5"
        />
      </span>
    );
  }

  return (
    <button
      type="button"
      className="flex items-center gap-1 text-xs px-2 py-0.5 rounded-full border border-dashed border-grayA-4 text-gray-8 hover:text-gray-10 hover:border-grayA-6 transition-all cursor-pointer w-full focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-grayA-6 focus-visible:ring-offset-1"
      onClick={(e) => {
        e.stopPropagation();
        onAdd(id);
      }}
    >
      <span className="flex-shrink-0">+</span>
      <span className="truncate capitalize">{slug}</span>
    </button>
  );
}
