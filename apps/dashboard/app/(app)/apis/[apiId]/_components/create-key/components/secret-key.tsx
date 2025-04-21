import { toast } from "@/components/ui/toaster";
import { cn } from "@/lib/utils";
import { CircleLock } from "@unkey/icons";
import { useState } from "react";

export const SecretKey = ({
  value,
  title = "Value",
  className,
}: {
  value: string;
  title: string;
  className?: string;
}) => {
  // Using hover state instead of clicked state
  const [isHovering, setIsHovering] = useState(false);

  // Show full value on hover, otherwise show partial with dots
  const displayValue = isHovering
    ? value
    : value.slice(0, 4) + "•".repeat(Math.max(80, value.length - 4));

  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    navigator.clipboard
      .writeText(value)
      .then(() => {
        toast.success(`${title} copied to clipboard`);
      })
      .catch((error) => {
        console.error("Failed to copy to clipboard:", error);
        toast.error("Failed to copy to clipboard");
      });
  };

  return (
    // biome-ignore lint/a11y/useKeyWithClickEvents: <explanation>
    <div
      className={cn(
        "rounded-xl border bg-white dark:bg-base-11 border-accent-4 text-grayA-11 w-full px-3 py-2 flex items-center cursor-pointer group relative",
        className,
      )}
      onClick={(e) => handleClick(e)}
      onMouseEnter={() => setIsHovering(true)}
      onMouseLeave={() => setIsHovering(false)}
    >
      <div className="flex items-center gap-2 flex-grow truncate pr-10">
        <CircleLock size="sm-regular" className="text-gray-12 flex-shrink-0" />
        <div className="truncate text-grayA-12 text-[13px]">{displayValue}</div>
      </div>
    </div>
  );
};
