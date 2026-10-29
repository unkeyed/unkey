import { StickySlot } from "@unkey/ui";
import { cn } from "cn";

export function OverviewChartFrame({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <StickySlot
      {...props}
      slot="chart"
      className={cn("w-full md:h-[clamp(12rem,30dvh,20rem)]", className)}
    />
  );
}
