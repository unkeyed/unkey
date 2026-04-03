"use client";

import { cn } from "@/lib/utils";
import type { SentinelPolicy } from "@/lib/trpc/routers/deploy/environment-settings/sentinel/update-middleware";
import { ChevronDown } from "@unkey/icons";
import { Badge, Button, FormInput } from "@unkey/ui";
import { Reorder, useDragControls } from "framer-motion";
import { type PointerEvent, useCallback, useState } from "react";

type SentinelPolicyRowProps = {
  policy: SentinelPolicy;
  index: number;
  isLast: boolean;
  onToggleActive: (id: string) => void;
  onUpdate: (id: string, field: "name", value: string) => void;
};

const POLICY_TYPE_LABELS: Record<SentinelPolicy["type"], string> = {
  keyauth: "Key Auth",
  ratelimit: "Rate Limit",
};


export function SentinelPolicyRow({
  policy,
  index,
  isLast,
  onToggleActive,
  onUpdate,
}: SentinelPolicyRowProps) {
  const controls = useDragControls();
  const [isOpen, setIsOpen] = useState(false);

  const handlePointerDown = useCallback(
    (e: PointerEvent) => {
      controls.start(e);
    },
    [controls],
  );

  const handleToggle = useCallback(() => {
    onToggleActive(policy.id);
  }, [onToggleActive, policy.id]);

  const handleRowClick = useCallback(() => {
    setIsOpen((prev) => !prev);
  }, []);

  return (
    <Reorder.Item
      value={policy}
      dragListener={false}
      dragControls={controls}
      transition={{ layout: { duration: 0 } }}
      className={cn(!isLast && "border-b border-grayA-4")}
    >
      <div className={cn(!policy.enabled && "opacity-55")}>
        <div
          role="button"
          tabIndex={0}
          aria-expanded={isOpen}
          className="group flex items-center hover:bg-grayA-2 transition-colors cursor-pointer"
          onClick={handleRowClick}
          onKeyDown={(e) => {
            if (e.key === "Enter" || e.key === " ") {
              e.preventDefault();
              handleRowClick();
            }
          }}
        >
          {/* Step number — same slot as checkbox: pl-4, w-8, shrink-0 */}
          <div className="pl-4 flex items-center shrink-0">
            <div
              className="size-6 rounded-full border border-grayA-5 bg-grayA-2 flex items-center justify-center text-[11px] font-medium text-gray-10"
            >
              {index + 1}
            </div>
          </div>

          {/* Drag handle */}
          <div
            className="flex items-center shrink-0 px-4 cursor-grab active:cursor-grabbing touch-none"
            onPointerDown={handlePointerDown}
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex flex-col gap-[3px] opacity-30 hover:opacity-60 p-0.5">
              <span className="block w-[13px] h-[1.5px] bg-gray-12 rounded-sm" />
              <span className="block w-[13px] h-[1.5px] bg-gray-12 rounded-sm" />
              <span className="block w-[13px] h-[1.5px] bg-gray-12 rounded-sm" />
            </div>
          </div>

          {/* Name cell — flex-4, py-3.5 (matches env-var name cell) */}
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

          {/* Type badge — flex-4, py-3.5, pr-3 (matches env-var value cell) */}
          <div className="flex-4 min-w-0 py-3.5 flex items-center pr-3">
            <Badge
              size="sm"
              className={cn(
                "rounded-md h-5.5 font-medium",
                policy.type === "keyauth"
                  ? "bg-infoA-3 text-infoA-11 border-infoA-5"
                  : "bg-warningA-3 text-warningA-11 border-warningA-5",
              )}
            >
              {POLICY_TYPE_LABELS[policy.type]}
            </Badge>
          </div>

          {/* Chevron + Active/Inactive toggle — w-auto, shrink-0, pr-4 */}
          <div className="shrink-0 py-3.5 flex items-center gap-2 pr-4">
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
            <ChevronDown
              iconSize="sm-regular"
              className={cn("text-gray-9 transition-transform", isOpen && "rotate-180")}
            />
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
    </Reorder.Item>
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
