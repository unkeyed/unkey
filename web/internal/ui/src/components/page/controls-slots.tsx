"use client";
import { type ReactNode, createContext, use, useCallback, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { cn } from "../../lib/utils";
import { isDev, useWarnIfMissing } from "./dev-warn";

type ControlsSlots = {
  end: HTMLElement | null;
  setEnd: (element: HTMLElement | null) => void;
};

const ControlsSlotsContext = createContext<ControlsSlots | null>(null);

export function ControlsSlotProvider({ children }: { children: ReactNode }) {
  const [end, setEnd] = useSlotElement("ControlsEndSlot");
  const value = useMemo(() => ({ end, setEnd }), [end, setEnd]);
  return <ControlsSlotsContext.Provider value={value}>{children}</ControlsSlotsContext.Provider>;
}

function useSlotElement(slot: string) {
  const [element, setElement] = useState<HTMLElement | null>(null);
  const registeredRef = useRef<HTMLElement | null>(null);
  const register = useCallback(
    (next: HTMLElement | null) => {
      const registered = registeredRef.current;
      if (isDev && next && registered?.isConnected && registered !== next) {
        console.warn(
          `Two ${slot} elements are mounted under one ControlsSlotProvider. Portal content shows only in the last one.`,
        );
      }
      registeredRef.current = next;
      setElement(next);
    },
    [slot],
  );
  return [element, register] as const;
}

export function ControlsEndSlot({ className }: { className?: string }) {
  const slots = use(ControlsSlotsContext);
  return (
    <div
      ref={slots?.setEnd}
      data-slot="controls-end"
      className={cn("flex shrink-0 items-center empty:hidden", className)}
    />
  );
}

export function ControlsEndPortal({ children }: { children: ReactNode }) {
  const target = use(ControlsSlotsContext)?.end ?? null;
  useWarnIfMissing(
    target !== null,
    "ControlsEndPortal has no mounted ControlsEndSlot, so its content is not shown.",
  );
  return target ? createPortal(children, target) : null;
}
