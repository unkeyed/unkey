import { StickySlot } from "@unkey/ui";

export function ControlsContainer({ children }: { children: React.ReactNode }) {
  return (
    <StickySlot slot="controls" className="flex flex-col border-b">
      <div className="flex items-center justify-between w-full px-3 py-1 min-h-10">{children}</div>
    </StickySlot>
  );
}

export function ControlsLeft({ children }: { children: React.ReactNode }) {
  return <div className="flex gap-2">{children}</div>;
}

export function ControlsRight({ children }: { children: React.ReactNode }) {
  return <div className="flex gap-2">{children}</div>;
}
