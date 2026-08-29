"use client";

import { cn } from "@/lib/utils";
import { Combobox as ComboboxPrimitive } from "@base-ui/react/combobox";
import { Check, ChevronExpandY, XMark } from "@unkey/icons";
import * as React from "react";

/**
 * Multi-select combobox built on Base UI's Combobox primitive, following
 * shadcn's "multiple" pattern: selected values render as removable chips
 * inside the trigger surface alongside an inline filter input.
 *
 * For single-select, use `Combobox` from `@unkey/ui` instead.
 */
export function Multibox<Value>(
  props: Omit<ComboboxPrimitive.Root.Props<Value, true>, "multiple">,
) {
  return <ComboboxPrimitive.Root openOnInputClick {...props} multiple />;
}

/**
 * Anchors the popup to the chips container instead of the default (the inline
 * input), which shifts as chips wrap. Pass the ref to both `MultiboxChips`
 * (`ref`) and `MultiboxContent` (`anchor`).
 */
export function useMultiboxAnchor() {
  return React.useRef<HTMLDivElement | null>(null);
}

export function MultiboxChips({ className, ...props }: ComboboxPrimitive.Chips.Props) {
  return (
    <ComboboxPrimitive.Chips
      className={cn(
        "relative flex min-h-9 w-full flex-wrap items-center gap-1.5 rounded-lg border border-gray-5 bg-white px-1.5 py-1 pr-8 text-[13px] leading-5 text-grayA-12 transition-colors duration-300 dark:bg-black",
        "hover:border-gray-8",
        "focus-within:border-accent-12 focus-within:ring-2 focus-within:ring-gray-5",
        "data-disabled:cursor-not-allowed data-disabled:opacity-50",
        className,
      )}
      {...props}
    />
  );
}

export function MultiboxChip({ className, ...props }: ComboboxPrimitive.Chip.Props) {
  return (
    <ComboboxPrimitive.Chip
      className={cn(
        "flex items-center gap-1 rounded-md border border-grayA-4 bg-grayA-3 px-1.5 py-0.5 text-xs text-accent-12",
        className,
      )}
      {...props}
    />
  );
}

export function MultiboxChipRemove({
  className,
  children,
  ...props
}: ComboboxPrimitive.ChipRemove.Props) {
  return (
    <ComboboxPrimitive.ChipRemove
      className={cn(
        "rounded p-0.5 text-grayA-9 transition-colors hover:bg-grayA-4 hover:text-accent-12",
        className,
      )}
      aria-label="Remove"
      {...props}
    >
      {children ?? <XMark iconSize="sm-regular" />}
    </ComboboxPrimitive.ChipRemove>
  );
}

export function MultiboxInput({ className, ...props }: ComboboxPrimitive.Input.Props) {
  return (
    <ComboboxPrimitive.Input
      className={cn(
        "h-6 min-w-12 flex-1 bg-transparent outline-hidden placeholder:text-grayA-8",
        className,
      )}
      {...props}
    />
  );
}

export function MultiboxTrigger({
  className,
  children,
  ...props
}: ComboboxPrimitive.Trigger.Props) {
  return (
    <ComboboxPrimitive.Trigger
      className={cn(
        "absolute right-2 top-1/2 flex -translate-y-1/2 items-center text-grayA-9",
        className,
      )}
      aria-label="Open list"
      {...props}
    >
      {children ?? <ChevronExpandY iconSize="sm-regular" />}
    </ComboboxPrimitive.Trigger>
  );
}

export function MultiboxContent({
  className,
  children,
  sideOffset = 4,
  ...props
}: ComboboxPrimitive.Positioner.Props) {
  return (
    <ComboboxPrimitive.Portal>
      <ComboboxPrimitive.Positioner className="isolate z-200" sideOffset={sideOffset} {...props}>
        <ComboboxPrimitive.Popup
          className={cn(
            "max-h-[min(var(--available-height),300px)] w-(--anchor-width) overflow-y-auto overflow-x-hidden rounded-lg border border-grayA-4 bg-white p-1 shadow-md scrollbar-thin dark:bg-black",
            className,
          )}
        >
          {children}
        </ComboboxPrimitive.Popup>
      </ComboboxPrimitive.Positioner>
    </ComboboxPrimitive.Portal>
  );
}

export function MultiboxEmpty({ className, ...props }: ComboboxPrimitive.Empty.Props) {
  return (
    <ComboboxPrimitive.Empty
      className={cn("py-6 text-center text-[13px] text-grayA-9 empty:p-0", className)}
      {...props}
    />
  );
}

export function MultiboxList(props: ComboboxPrimitive.List.Props) {
  return <ComboboxPrimitive.List {...props} />;
}

export function MultiboxItem({ className, children, ...props }: ComboboxPrimitive.Item.Props) {
  return (
    <ComboboxPrimitive.Item
      className={cn(
        "relative flex cursor-pointer select-none items-center gap-2 rounded-sm px-2 py-1.5 text-[13px] text-gray-12 outline-hidden",
        "data-highlighted:bg-grayA-3 data-highlighted:text-grayA-12",
        "data-disabled:pointer-events-none data-disabled:opacity-50",
        className,
      )}
      {...props}
    >
      {children}
      <ComboboxPrimitive.ItemIndicator className="ml-auto shrink-0">
        <Check iconSize="sm-regular" />
      </ComboboxPrimitive.ItemIndicator>
    </ComboboxPrimitive.Item>
  );
}
