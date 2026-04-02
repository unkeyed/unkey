import { cn } from "@/lib/utils";
import { ChartActivity2 } from "@unkey/icons";
import { Badge, Checkbox, TimestampInfo } from "@unkey/ui";
import type { ReactNode } from "react";

type EnvVarBaseRowProps = {
  nameCell: ReactNode;
  valueCell: ReactNode;
  timestamp: number;
  actionsCell?: ReactNode;
  expandedContent?: ReactNode;
  showCheckbox: boolean;
  checked: boolean | "indeterminate";
  forceCheckboxVisible: boolean;
  onCheckboxClick?: (shiftKey: boolean) => void;
  onRowClick?: () => void;
};

export function EnvVarBaseRow({
  nameCell,
  valueCell,
  timestamp,
  actionsCell,
  expandedContent,
  showCheckbox,
  checked,
  forceCheckboxVisible,
  onCheckboxClick,
  onRowClick,
}: EnvVarBaseRowProps) {
  const isClickable = !!onRowClick;

  const rowProps = isClickable
    ? {
        role: "button" as const,
        tabIndex: 0,
        onClick: onRowClick,
        onKeyDown: (e: React.KeyboardEvent) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            onRowClick();
          }
        },
      }
    : {};

  return (
    <div>
      <div
        {...rowProps}
        className={cn(
          "group flex items-center hover:bg-grayA-2 transition-colors",
          isClickable && "cursor-pointer",
        )}
      >
        {/* biome-ignore lint/a11y/useKeyWithClickEvents: checkbox handles keyboard interaction */}
        <div
          className="pl-4 flex items-center w-8 shrink-0"
          onClick={(e) => {
            if (showCheckbox && onCheckboxClick) {
              e.stopPropagation();
              onCheckboxClick(e.shiftKey);
            }
          }}
        >
          {showCheckbox && (
            <Checkbox
              checked={checked}
              className={cn(
                "size-4 [&_svg]:size-3",
                forceCheckboxVisible
                  ? "opacity-100"
                  : "opacity-0 pointer-events-none group-hover:opacity-100 group-hover:pointer-events-auto focus-visible:opacity-100 focus-visible:pointer-events-auto",
              )}
              onCheckedChange={() => {}}
            />
          )}
        </div>
        <div className="flex-4 min-w-0 py-3.5 flex items-center">{nameCell}</div>
        <div className="flex-4 min-w-0 py-3.5 flex items-center pr-3">{valueCell}</div>
        <div className="flex-2 min-w-0 py-3.5 flex items-center pr-3">
          <TimestampBadge value={timestamp} />
        </div>
        <div
          className={cn(
            "w-12 shrink-0 py-3.5 flex items-center justify-end",
            actionsCell ? "pr-4" : "pr-3",
          )}
        >
          {actionsCell}
        </div>
      </div>
      {expandedContent && (
        <div className="grid animate-expand-down overflow-hidden">
          <div className="min-h-0">{expandedContent}</div>
        </div>
      )}
    </div>
  );
}

export function TimestampBadge({ value }: { value: number }) {
  return (
    <Badge className="px-1.5 rounded-md flex gap-2 items-center h-5.5 border-none bg-grayA-3 text-grayA-12 truncate">
      <ChartActivity2 iconSize="sm-regular" className="shrink-0" />
      <TimestampInfo displayType="relative" value={value} className="truncate" />
    </Badge>
  );
}
