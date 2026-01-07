import { cn } from "@/lib/utils";

type StatusBadgeProps = {
  primary: {
    label: string;
    color: string;
    icon: React.ReactNode;
  };
};

export const StatusBadge = ({ primary }: StatusBadgeProps) => {
  return (
    <div className="flex items-center justify-start gap-0.5 text-xs">
      <div
        className={cn(
          primary.color,
          "px-1.5 py-1 flex items-center justify-center gap-2 h-[22px]",
          "rounded-md",
        )}
      >
        {primary.icon && <span className="shrink-0">{primary.icon}</span>}
        <span>{primary.label}</span>
      </div>
    </div>
  );
};