import { cn } from "@/lib/utils";
import { ChartActivity2 } from "@unkey/icons";
import { Badge, TimestampInfo } from "@unkey/ui";
import { useRef, useState } from "react";
import { STATUS_STYLES } from "../utils/get-row-class";

export const LastUpdated = ({
  isSelected,
  lastUpdated,
}: {
  isSelected: boolean;
  lastUpdated: number;
}) => {
  const badgeRef = useRef<HTMLDivElement>(null);
  const [showTooltip, setShowTooltip] = useState(false);

  return (
    <Badge
      ref={badgeRef}
      className={cn(
        "px-1.5 rounded-md flex gap-2 items-center max-w-min h-[22px] border-none cursor-pointer",
        isSelected ? STATUS_STYLES.badge.selected : STATUS_STYLES.badge.default
      )}
      onMouseOver={() => {
        setShowTooltip(true);
      }}
      onMouseLeave={() => {
        setShowTooltip(false);
      }}
    >
      <div>
        <ChartActivity2 iconsize="sm-regular" />
      </div>
      <div className="truncate">
        <TimestampInfo
          displayType="relative"
          value={lastUpdated}
          className="truncate"
          triggerRef={badgeRef}
          open={showTooltip}
          onOpenChange={setShowTooltip}
        />
      </div>
    </Badge>
  );
};
