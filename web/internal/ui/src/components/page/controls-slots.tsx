"use client";
import {
  type ReactNode,
  createContext,
  use,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";
import { cn } from "../../lib/utils";

type ControlsSlots = {
  end: HTMLElement | null;
  selection: HTMLElement | null;
  setEnd: (element: HTMLElement | null) => void;
  setSelection: (element: HTMLElement | null) => void;
};

const ControlsSlotsContext = createContext<ControlsSlots | null>(null);

const isDev = typeof process !== "undefined" && process.env.NODE_ENV !== "production";

export function ControlsSlotProvider({ children }: { children: ReactNode }) {
  const [end, setEnd] = useSlotElement("ControlsEndSlot");
  const [selection, setSelection] = useSlotElement("ControlsSelectionSlot");
  const value = useMemo(
    () => ({ end, selection, setEnd, setSelection }),
    [end, selection, setEnd, setSelection],
  );
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

export function ControlsSelectionSlot({ className }: { className?: string }) {
  const slots = use(ControlsSlotsContext);
  return (
    <div
      ref={slots?.setSelection}
      data-slot="controls-selection"
      className={cn(
        "flex min-w-0 flex-1 items-center overflow-x-clip [overflow-clip-margin:4px] empty:hidden",
        className,
      )}
    />
  );
}

export function ControlsEndPortal({ children }: { children: ReactNode }) {
  const target = use(ControlsSlotsContext)?.end ?? null;
  useWarnIfNoTarget(target, "ControlsEndPortal", "ControlsEndSlot");
  return target ? createPortal(children, target) : null;
}

export function ControlsSelectionPortal({ children }: { children: ReactNode }) {
  const target = use(ControlsSlotsContext)?.selection ?? null;
  useWarnIfNoTarget(target, "ControlsSelectionPortal", "ControlsSelectionSlot");
  return target ? createPortal(children, target) : null;
}

function useWarnIfNoTarget(target: HTMLElement | null, portal: string, slot: string) {
  useEffect(() => {
    if (!isDev || target) {
      return;
    }
    return afterSlotRefsCommit(() =>
      console.warn(`${portal} has no mounted ${slot}, so its content is not shown.`),
    );
  }, [target, portal, slot]);
}

function afterSlotRefsCommit(callback: () => void) {
  const timer = setTimeout(callback, 0);
  return () => clearTimeout(timer);
}
