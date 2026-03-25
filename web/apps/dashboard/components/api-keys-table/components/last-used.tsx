import { cn } from "@/lib/utils";
import { ChartActivity2 } from "@unkey/icons";
import { Badge, TimestampInfo } from "@unkey/ui";
import { STATUS_STYLES } from "@unkey/ui";
import { useRef, useState } from "react";

export const LastUsedCell = ({
  lastUsedAt,
  isSelected,
}: {
  lastUsedAt: number;
  isSelected: boolean;
}) => {
  const badgeRef = useRef<HTMLElement>(null) as React.RefObject<HTMLElement>;
  const [showTooltip, setShowTooltip] = useState(false);

  return (
    <Badge
      ref={badgeRef}
      className={cn(
        "px-1.5 rounded-md flex gap-2 items-center max-w-min h-5.5 border-none cursor-pointer",
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
        {lastUsedAt > 0 ? (
          <TimestampInfo
            displayType="relative"
            value={lastUsedAt}
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
