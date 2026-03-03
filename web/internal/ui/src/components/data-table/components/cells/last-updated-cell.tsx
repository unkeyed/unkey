"use client";
import { ChartActivity2 } from "@unkey/icons";
import type React from "react";
import { useRef, useState } from "react";
import { cn } from "../../../../lib/utils";
import { Badge } from "../../../badge";
import { TimestampInfo } from "../../../timestamp-info";
import { STATUS_STYLES } from "../../constants/constants";

export interface LastUpdatedCellProps {
  isSelected: boolean;
  lastUpdated?: number | null;
}

export const LastUpdatedCell = ({ isSelected, lastUpdated }: LastUpdatedCellProps) => {
  const badgeRef = useRef<HTMLElement>(null) as React.RefObject<HTMLElement>;
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
      <div>
        <ChartActivity2 iconSize="sm-regular" />
      </div>
      <div className="truncate">
        {lastUpdated ? (
          <TimestampInfo
            displayType="relative"
            value={lastUpdated}
            className="truncate"
            triggerRef={badgeRef}
            open={showTooltip}
            onOpenChange={setShowTooltip}
          />
        ) : (
          "Never used"
        )}
      </div>
    </Badge>
  );
};
