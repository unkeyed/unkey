"use client";

import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import type { SentinelPolicy } from "@/lib/trpc/routers/deploy/environment-settings/sentinel/update-middleware";
import { cn } from "@/lib/utils";
import { Dots, GripDotsVertical, PenWriting3, Trash } from "@unkey/icons";
import { Button, ConfirmPopover, FormInput } from "@unkey/ui";
import { useCallback, useRef, useState } from "react";

type SentinelPolicyRowProps = {
  policy: SentinelPolicy;
  index: number;
  isLast: boolean;
  isDragOver: boolean;
  onToggleActive: (id: string) => void;
  onUpdate: (id: string, field: "name", value: string) => void;
  onDelete: (id: string) => void;
  onDragStart: (index: number) => void;
  onDragOver: (index: number) => void;
  onDrop: (index: number) => void;
  onDragEnd: () => void;
};

const POLICY_TYPE_LABELS: Record<SentinelPolicy["type"], string> = {
  keyauth: "Key Auth",
  ratelimit: "Rate Limit",
  jwt: "JWT Auth",
  basicauth: "Basic Auth",
  iprules: "IP Rules",
  openapi: "OpenAPI",
};

export function SentinelPolicyRow({
  policy,
  index,
  isLast,
  isDragOver,
  onToggleActive,
  onUpdate,
  onDelete,
  onDragStart,
  onDragOver,
  onDrop,
  onDragEnd,
}: SentinelPolicyRowProps) {
  const fromHandle = useRef(false);
  const deleteButtonRef = useRef<HTMLButtonElement>(null);
  const [isOpen, setIsOpen] = useState(false);
  const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false);

  const handleToggle = useCallback(() => {
    onToggleActive(policy.id);
  }, [onToggleActive, policy.id]);

  const menuItems: MenuItem[] = [
    {
      id: "edit",
      label: "Edit",
      icon: <PenWriting3 iconSize="md-regular" />,
      divider: true,
      onClick: (e) => {
        e.stopPropagation();
        setIsOpen(true);
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

  return (
    <>
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
        <div className={cn(!policy.enabled && "opacity-55")}>
          <div className="group flex items-center hover:bg-grayA-2 transition-colors">
            {/* Step number: fixed 40px column */}
            <div className="w-10 shrink-0  py-3.5 pl-4">
              <div
                className={cn(
                  "size-6 rounded-full border flex items-center justify-center text-[11px] font-medium",
                  policy.enabled
                    ? "bg-info-3 border-info-7 text-info-11"
                    : "bg-grayA-2 border-grayA-5 text-gray-10",
                )}
              >
                {index + 1}
              </div>
            </div>

            {/* Drag handle: fixed 40px column */}
            <div
              className="w-10 shrink-0 flex items-center justify-center py-3.5 cursor-grab active:cursor-grabbing touch-none"
              onMouseDown={() => {
                fromHandle.current = true;
              }}
              onClick={(e) => e.stopPropagation()}
            >
              <GripDotsVertical iconSize="lg-medium" className="opacity-30 hover:opacity-60" />
            </div>

            {/* Name: flex-4, matches env-var name cell */}
            <div className="flex-4 min-w-0 py-3.5 flex items-center">
              <span
                className={cn(
                  "text-[13px] truncate",
                  policy.name ? "text-gray-12" : "text-gray-9 italic",
                )}
              >
                {policy.name || "Untitled policy"}
              </span>
            </div>

            {/* Type: flex-2, plain muted text */}
            <div className="flex-4 min-w-0 py-3.5 flex items-center pr-3">
              <span className="text-[13px] text-gray-11 truncate">
                {POLICY_TYPE_LABELS[policy.type]}
              </span>
            </div>

            {/* Active/Inactive toggle + dots menu: fixed width so toggle text can't shift the name */}
            <div className="w-36 shrink-0 py-3.5 flex items-center justify-end gap-2 pr-4">
              <button
                type="button"
                aria-pressed={policy.enabled}
                className={cn(
                  "flex items-center gap-1.5 text-xs px-2 py-0.5 rounded-full border shrink-0 transition-all cursor-pointer",
                  "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-1",
                  policy.enabled
                    ? "bg-info-3 border-info-7 text-info-11 focus-visible:ring-info-7"
                    : "bg-transparent border-grayA-5 text-gray-10 focus-visible:ring-grayA-7",
                )}
                onClick={(e) => {
                  e.stopPropagation();
                  handleToggle();
                }}
              >
                <span
                  className={cn(
                    "w-1.5 h-1.5 rounded-full flex-shrink-0",
                    policy.enabled ? "bg-info-11" : "bg-gray-9",
                  )}
                />
                {policy.enabled ? "Active" : "Inactive"}
              </button>
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
            </div>
          </div>

          {/* Expandable edit — matches env-var-base-row expand pattern */}
          {isOpen && (
            <div className="grid animate-expand-down overflow-hidden">
              <div className="min-h-0">
                <SentinelPolicyEditBody
                  policy={policy}
                  onSave={(name) => {
                    onUpdate(policy.id, "name", name);
                    setIsOpen(false);
                  }}
                  onClose={() => setIsOpen(false)}
                />
              </div>
            </div>
          )}
        </div>
      </div>

      <ConfirmPopover
        isOpen={isDeleteConfirmOpen}
        onOpenChange={setIsDeleteConfirmOpen}
        onConfirm={() => onDelete(policy.id)}
        triggerRef={deleteButtonRef}
        title="Confirm deletion"
        description={`This will permanently delete "${policy.name || "this policy"}". This action cannot be undone.`}
        confirmButtonText="Delete policy"
        cancelButtonText="Cancel"
        variant="danger"
      />
    </>
  );
}

function SentinelPolicyEditBody({
  policy,
  onSave,
  onClose,
}: {
  policy: SentinelPolicy;
  onSave: (name: string) => void;
  onClose: () => void;
}) {
  const [draft, setDraft] = useState(policy.name);

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      onSave(draft);
    },
    [draft, onSave],
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        onClose();
      }
    },
    [onClose],
  );

  return (
    <div className="bg-gray-1 px-4 pb-6 pt-5 border-t border-grayA-4" onKeyDown={handleKeyDown}>
      <form onSubmit={handleSubmit} className="flex flex-col gap-5">
        <FormInput
          label="Name"
          placeholder="Policy name"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
        />
        <div className="flex items-center justify-end gap-2 pt-5 mt-1">
          <Button type="button" variant="outline" size="md" onClick={onClose} className="px-3">
            Cancel
          </Button>
          <Button type="submit" variant="primary" size="md" className="px-3">
            Save
          </Button>
        </div>
      </form>
    </div>
  );
}
