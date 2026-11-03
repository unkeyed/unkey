import { ControlsEndSlot, ControlsSelectionSlot, StickySlot } from "@unkey/ui";
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
      <div className="group/controls @container/controls flex items-center gap-2 w-full px-3 py-1 min-h-10">
        <div className="flex min-w-0 flex-1 items-center gap-2">
          <ControlsSelectionSlot className="order-first" />
          {children}
        </div>
        <ControlsEndSlot />
      </div>
    </StickySlot>
  );
}

export function ControlsLeft({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-w-0 gap-2 overflow-x-clip [overflow-clip-margin:4px] group-has-[[data-slot=controls-selection]:not(:empty)]/controls:hidden">
      {children}
    </div>
  );
}

export function ControlsRight({ children }: { children: React.ReactNode }) {
  return <div className="ml-auto flex shrink-0 gap-2">{children}</div>;
}
