"use client";

import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { Switch } from "@/components/ui/switch";
import type { Policy } from "@/lib/collections/deploy/policies.schema";
import { cn } from "@/lib/utils";
import { Dots, GripDotsVertical, PenWriting3, Trash } from "@unkey/icons";
import { Button, ConfirmPopover } from "@unkey/ui";
import { useRef, useState } from "react";

type MergedPolicyRow = {
  key: string;
  name: string;
  type: Policy["type"];
  production: Policy | null;
  preview: Policy | null;
};

type PolicyRowProps = {
  policy: MergedPolicyRow;
  index: number;
  isLast: boolean;
  isDragOver: boolean;
  productionSlug: string;
  previewSlug: string;
  onToggleProduction: (key: string) => void;
  onTogglePreview: (key: string) => void;
  onAddToProduction: (key: string) => void;
  onAddToPreview: (key: string) => void;
  onDelete: (key: string) => void;
  onEdit: (policy: Policy) => void;
  onDragStart: (index: number) => void;
  onDragOver: (index: number) => void;
  onDrop: (index: number) => void;
  onDragEnd: () => void;
};

const POLICY_TYPE_LABELS: Record<Policy["type"], string> = {
  keyauth: "Key Auth",
  ratelimit: "Rate Limit",
  firewall: "Firewall",
  openapi: "OpenAPI Validation",
  logging: "Logging",
};

export function PolicyRow({
  policy,
  index,
  isLast,
  isDragOver,
  productionSlug,
  previewSlug,
  onToggleProduction,
  onTogglePreview,
  onAddToProduction,
  onAddToPreview,
  onDelete,
  onEdit,
  onDragStart,
  onDragOver,
  onDrop,
  onDragEnd,
}: PolicyRowProps) {
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
        const target = policy.production ?? policy.preview;
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

  const isActiveAnywhere =
    (policy.production?.enabled ?? false) || (policy.preview?.enabled ?? false);

  return (
    <div
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
      className={cn(
        // If text under the pointer is selected, the browser drags the
        // selection and not this row. It then shows a large page area.
        "select-none",
        !isLast && "border-b border-grayA-4",
        isDragOver && "bg-grayA-3",
      )}
    >
      <div className={cn(!isActiveAnywhere && "opacity-55")}>
        {/* biome-ignore lint/a11y/useSemanticElements: intentionally a div (not a native button) so the nested drag-handle and action buttons remain valid HTML */}
        <div
          role="button"
          tabIndex={0}
          className="group flex items-center hover:bg-grayA-2 transition-colors cursor-pointer w-full text-left"
          onClick={() => {
            const target = policy.production ?? policy.preview;
            if (target) {
              onEdit(target);
            }
          }}
          onKeyDown={(e) => {
            if (e.key === "Enter" || e.key === " ") {
              e.preventDefault();
              const target = policy.production ?? policy.preview;
              if (target) {
                onEdit(target);
              }
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
              policyKey={policy.key}
              slug={productionSlug}
              envPolicy={policy.production}
              onToggle={onToggleProduction}
              onAdd={onAddToProduction}
            />
            <EnvSwitch
              policyKey={policy.key}
              slug={previewSlug}
              envPolicy={policy.preview}
              onToggle={onTogglePreview}
              onAdd={onAddToPreview}
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
              onConfirm={() => onDelete(policy.key)}
              triggerRef={deleteButtonRef}
              title="Confirm deletion"
              description={`This will permanently delete "${policy.name}". This action cannot be undone.`}
              confirmButtonText="Delete policy"
              cancelButtonText="Cancel"
              variant="danger"
            />
          </div>
        </div>
      </div>
    </div>
  );
}

function EnvSwitch({
  policyKey,
  slug,
  envPolicy,
  onToggle,
  onAdd,
}: {
  policyKey: string;
  slug: string;
  envPolicy: Policy | null;
  onToggle: (key: string) => void;
  onAdd: (key: string) => void;
}) {
  if (envPolicy !== null) {
    return (
      <span
        className="flex items-center gap-2 shrink-0"
        onClick={(e) => e.stopPropagation()}
        onKeyDown={(e) => e.stopPropagation()}
      >
        <span className="text-[13px] text-gray-11 capitalize whitespace-nowrap">{slug}</span>
        <Switch checked={envPolicy.enabled} onCheckedChange={() => onToggle(policyKey)} size="sm" />
      </span>
    );
  }

  return (
    <button
      type="button"
      className="flex items-center gap-1 text-xs px-2 py-0.5 rounded-full border border-dashed border-grayA-4 text-gray-8 hover:text-gray-10 hover:border-grayA-6 transition-all cursor-pointer w-full focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-grayA-6 focus-visible:ring-offset-1"
      onClick={(e) => {
        e.stopPropagation();
        onAdd(policyKey);
      }}
    >
      <span className="flex-shrink-0">+</span>
      <span className="truncate capitalize">{slug}</span>
    </button>
  );
}

/**
 * Sets the drag image to a detached copy of the row.
 *
 * The browser selects the drag image. For this list it captures a large page
 * area and not the row. The live row does not work as the drag image, because
 * React replaces that node while the drag runs. A copy on `document.body` is
 * outside the render tree, so nothing replaces it.
 */
function setRowDragImage(e: React.DragEvent<HTMLDivElement>) {
  const source = e.currentTarget;
  const { width, height } = source.getBoundingClientRect();
  const clone = source.cloneNode(true) as HTMLElement;

  // A row draws only its bottom divider. The list container draws the frame
  // and the corners. The copy is outside that container, so give the copy a
  // frame and a background.
  clone.classList.remove("border-b");
  clone.classList.add("border", "border-grayA-4", "rounded-lg", "bg-gray-1", "shadow-lg");

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
