import { InfoTooltip } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import type { PropsWithChildren } from "react";

type DisabledWrapperProps = PropsWithChildren<{
  disabled?: boolean;
  tooltipContent?: string;
  className?: string;
}>;

export const DisabledWrapper = ({
  disabled = true,
  tooltipContent = "Coming soon",
  className,
  children,
}: DisabledWrapperProps) => {
  if (!disabled) {
    return <>{children}</>;
  }

  return (
    <InfoTooltip
      asChild
      variant="muted"
      content={tooltipContent}
      className="px-2.5 py-1 rounded-[10px] text-whiteA-12 bg-blackA-12 text-xs z-30"
      position={{ align: "center", side: "top", sideOffset: 5 }}
    >
      <div
        className={cn(
          "grayscale opacity-40 cursor-not-allowed relative border border-dashed border-grayA-6 rounded-md p-1",
          className,
        )}
      >
        <div className="absolute inset-0 bg-black/15 rounded-md pointer-events-none" />
        <div className={cn("relative", className)}>{children}</div>
      </div>
    </InfoTooltip>
  );
};
