import { cn } from "@/lib/utils";

type StatusBadgeProps = {
  // Accept simplified primary status info
  primary: {
    label: string;
    color: string;
    icon: React.ReactNode;
  };
  count: number;
};

export const StatusBadge = ({ primary, count }: StatusBadgeProps) => {
  return (
    <div className="flex items-center justify-start gap-0.5 text-xs font-medium">
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
      {count > 0 && (
        <div
          className={cn(
            primary.color,
            "rounded-r-md px-1.5 py-1 flex items-center justify-center h-[22px]",
          )}
        >
          +{count}
        </div>
      )}
    </div>
  );
};
