import { cn } from "@/lib/utils";
import { Badge } from "@unkey/ui";
import { useRef } from "react";
import { STATUS_STYLES } from "../../../utils/get-row-class";

type StatusBadgeProps = {
  primary: {
    label: string;
    color: string;
    icon: React.ReactNode;
  };
  count: number;
  isSelected?: boolean;
};

export const StatusBadge = ({ primary, count, isSelected = false }: StatusBadgeProps) => {
  const badgeRef = useRef<HTMLDivElement>(null);

  const isDisabled = primary.label === "Disabled";

  return (
    <div className="flex items-center justify-start gap-0.5 text-xs">
      {isDisabled ? (
        // Use Badge component only for "Disabled" label
        <Badge
          ref={badgeRef}
          className={cn(
            "px-1.5 rounded-md flex gap-2 items-center max-w-min h-[22px] border-none cursor-pointer",
            isSelected ? STATUS_STYLES.badge.selected : STATUS_STYLES.badge.default,
          )}
        >
          {primary.icon && <span>{primary.icon}</span>}
          <span>{primary.label}</span>
        </Badge>
      ) : (
        <div
          className={cn(
            primary.color,
            "px-1.5 py-1 flex items-center justify-center gap-2 h-[22px]",
            count > 0 ? "rounded-l-md" : "rounded-md",
          )}
        >
          {primary.icon && <span>{primary.icon}</span>}
          <span>{primary.label}</span>
        </div>
      )}

      {count > 0 &&
        (isDisabled ? (
          <Badge
            className={cn(
              "rounded-r-md px-1.5 py-1 flex items-center justify-center h-[22px] border-none",
              primary.color,
            )}
          >
            +{count}
          </Badge>
        ) : (
          <div
            className={cn(
              primary.color,
              "rounded-r-md px-1.5 py-1 flex items-center justify-center h-[22px]",
            )}
          >
            +{count}
          </div>
        ))}
    </div>
  );
};
