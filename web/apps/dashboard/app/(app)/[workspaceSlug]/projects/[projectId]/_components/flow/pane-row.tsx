"use client";

import {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemMedia,
  ItemTitle,
  cn,
} from "@unkey/ui";
import type { ReactNode } from "react";

type PaneRowProps = {
  icon: ReactNode;
  label: ReactNode;
  hint?: ReactNode;
  trailing?: ReactNode;
  selected?: boolean;
  disabled?: boolean;
  variant?: "default" | "outline";
  onClick: () => void;
};

const VARIANT_CLASS: Record<NonNullable<PaneRowProps["variant"]>, string> = {
  default: "rounded-none disabled:hover:bg-transparent",
  outline: "bg-raised disabled:hover:bg-raised data-[selected=true]:border-grayA-8",
};

export function PaneRow({
  icon,
  label,
  hint,
  trailing,
  selected = false,
  disabled = false,
  variant = "default",
  onClick,
}: PaneRowProps) {
  return (
    <Item
      variant={variant}
      data-selected={selected}
      className={cn(
        "disabled:cursor-not-allowed disabled:opacity-50 data-[selected=true]:bg-grayA-2",
        VARIANT_CLASS[variant],
      )}
      render={
        <button
          type="button"
          onClick={onClick}
          disabled={disabled}
          aria-current={selected || undefined}
        >
          <ItemMedia>{icon}</ItemMedia>
          <ItemContent>
            <ItemTitle>{label}</ItemTitle>
            {hint ? <ItemDescription>{hint}</ItemDescription> : null}
          </ItemContent>
          {trailing ? <ItemActions>{trailing}</ItemActions> : null}
        </button>
      }
    />
  );
}
