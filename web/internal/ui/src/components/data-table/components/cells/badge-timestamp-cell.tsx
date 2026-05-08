"use client";
// biome-ignore lint/correctness/noUnusedImports: React is needed for JSX
import React from "react";
import type { ReactNode } from "react";
import { useRef, useState } from "react";
import { cn } from "../../../../lib/utils";
import { Badge } from "../../../badge";
import { TimestampInfo } from "../../../timestamp-info";
import { STATUS_STYLES } from "../../constants/constants";

export interface BadgeTimestampCellProps {
  isSelected: boolean;
  timestamp?: number | null;
  icon: ReactNode;
  emptyText: string;
}

export const BadgeTimestampCell = ({
  isSelected,
  timestamp,
  icon,
  emptyText,
}: BadgeTimestampCellProps) => {
  const badgeRef = useRef<HTMLSpanElement>(null);
  const [showTooltip, setShowTooltip] = useState(false);

  return (
    <Badge
      ref={badgeRef}
      className={cn(
        "px-1.5 rounded-md flex gap-2 items-center max-w-min h-[22px] border-none cursor-pointer",
        isSelected ? STATUS_STYLES.badge.selected : STATUS_STYLES.badge.default,
      )}
      onMouseOver={() => {
        setShowTooltip(true);
      }}
      onMouseLeave={() => {
        setShowTooltip(false);
      }}
    >
      <div>{icon}</div>
      <div className="truncate">
        {timestamp != null ? (
          <TimestampInfo
            displayType="relative"
            value={timestamp}
            className="truncate"
            triggerRef={badgeRef}
            open={showTooltip}
            onOpenChange={setShowTooltip}
          />
        ) : (
          emptyText
        )}
      </div>
    </Badge>
  );
};
