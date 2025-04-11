import { cn } from "@unkey/ui/src/lib/utils";

type SettingCardProps = {
  title: string | React.ReactNode;
  description: string | React.ReactNode;
  children?: React.ReactNode;
  className?: string;
  border?: "top" | "bottom" | "both" | "none" | "default";
  contentWidth?: string;
};

export function SettingCard({
  title,
  description,
  children,
  className,
  border = "default",
  contentWidth = "w-[320px]",
}: SettingCardProps) {
  const borderRadiusClass = {
    "rounded-t-xl": border === "top",
    "rounded-b-xl": border === "bottom",
    "rounded-xl": border === "both",
    "": border === "none" || border === "default",
  };

  const borderClass = {
    "border border-gray-4": border !== "none",
    "border-t-0": border === "bottom",
    "border-b-0": border === "top",
  };

  return (
    <div
      className={cn(
        "px-6 py-3 w-full flex gap-6 justify-between items-center",
        borderRadiusClass,
        borderClass,
        className,
      )}
    >
      <div className="flex flex-col gap-1 text-sm">
        <div className="font-medium text-accent-12">{title}</div>
        <div className="text-accent-11">{description}</div>
      </div>
      <div className={cn("flex items-center", contentWidth)}>{children}</div>
    </div>
  );
}
