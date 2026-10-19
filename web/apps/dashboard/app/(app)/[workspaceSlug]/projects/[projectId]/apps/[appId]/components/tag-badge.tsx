import { cn } from "cn";
import type { HTMLAttributes } from "react";

type TagBadgeProps = HTMLAttributes<HTMLDivElement> & {
  text: string;
};

export function TagBadge({ text, className, ...props }: TagBadgeProps) {
  return (
    <div
      className={cn(
        "inline-flex items-center rounded-md px-2 text-xs leading-5 gap-1 shrink-0 text-featureA-11 bg-featureA-3",
        className,
      )}
      {...props}
    >
      <span className="font-medium">{text}</span>
    </div>
  );
}
