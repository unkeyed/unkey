import { ControlsEndSlot, StickySlot } from "@unkey/ui";
import { cn } from "cn";

export function ControlsContainer({ children }: { children?: React.ReactNode }) {
  return (
    <StickySlot
      slot="controls"
      className={cn(
        "flex flex-col border-b",
        !children && "has-[[data-slot=controls-end]:empty]:hidden",
      )}
    >
      <div className="@container/controls flex items-center gap-2 w-full px-3 py-1 min-h-10">
        <div className="flex min-w-0 flex-1 items-center gap-2">{children}</div>
        <ControlsEndSlot />
      </div>
    </StickySlot>
  );
}

export function ControlsLeft({ children }: { children: React.ReactNode }) {
  return <div className="flex -mx-1 min-w-0 gap-2 overflow-x-clip px-1">{children}</div>;
}

export function ControlsRight({ children }: { children: React.ReactNode }) {
  return <div className="ml-auto flex shrink-0 gap-2">{children}</div>;
}
