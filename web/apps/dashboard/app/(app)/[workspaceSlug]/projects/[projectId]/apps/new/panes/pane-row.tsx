"use client";

import { Item, ItemActions, ItemContent, ItemDescription, ItemMedia, ItemTitle } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import type { ReactNode } from "react";

type PaneRowProps = {
  icon: ReactNode;
  label: ReactNode;
  hint?: ReactNode;
  trailing?: ReactNode;
  selected?: boolean;
  disabled?: boolean;
  onClick: () => void;
};

export function PaneRow({
  icon,
  label,
  hint,
  trailing,
  selected = false,
  disabled = false,
  onClick,
}: PaneRowProps) {
  return (
    <Item
      className={cn(
        "rounded-none disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:bg-transparent",
        selected && "bg-grayA-2",
      )}
      render={
        <button type="button" onClick={onClick} disabled={disabled} aria-pressed={selected}>
          <ItemMedia>{icon}</ItemMedia>
          <ItemContent>
            <ItemTitle className="truncate">{label}</ItemTitle>
            {hint ? <ItemDescription className="truncate">{hint}</ItemDescription> : null}
          </ItemContent>
          {trailing ? <ItemActions>{trailing}</ItemActions> : null}
        </button>
      }
    />
  );
}
