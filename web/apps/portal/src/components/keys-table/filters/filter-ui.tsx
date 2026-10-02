import { Activity, CircleDot, KeyRound, type LucideIcon, X } from "lucide-react";
import { useId } from "react";
import { formatCount } from "~/components/analytics/format";
import { Badge } from "~/components/ui/badge";
import { Checkbox } from "~/components/ui/checkbox";
import { Tooltip, TooltipContent, TooltipTrigger } from "~/components/ui/tooltip";
import type { AppliedFilter, Dimension } from "./filter-model";

export const DIMENSION_ICONS: Record<Dimension, LucideIcon> = {
  keys: KeyRound,
  outcomes: Activity,
  status: CircleDot,
};

export function Kbd({ children }: { children: React.ReactNode }) {
  return (
    <Badge
      variant="secondary"
      className="rounded px-1.5 py-0 font-mono font-normal text-[10px] text-gray-10"
    >
      {children}
    </Badge>
  );
}

export function FilterPill({ dimension, label, value, onRemove }: AppliedFilter) {
  const Icon = DIMENSION_ICONS[dimension];

  return (
    <span className="inline-flex h-8 max-w-full items-stretch overflow-hidden rounded-md border border-gray-5 bg-gray-2 font-medium text-gray-11 text-xs shadow-xs">
      <Tooltip>
        <TooltipTrigger
          aria-label={label}
          className="flex shrink-0 items-center border-gray-5 border-r px-2 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-12 focus-visible:ring-inset"
        >
          <Icon className="size-3.5 text-gray-10" aria-hidden="true" />
        </TooltipTrigger>
        <TooltipContent side="top">{label}</TooltipContent>
      </Tooltip>
      <span className="flex min-w-0 items-center px-2" title={value}>
        <span className="max-w-64 truncate">{value}</span>
      </span>
      <button
        type="button"
        onClick={onRemove}
        aria-label={`Remove filter ${label}: ${value}`}
        className="flex w-7 shrink-0 items-center justify-center text-gray-10 transition-colors hover:bg-gray-3 hover:text-gray-12 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-12 focus-visible:ring-inset"
      >
        <X className="size-3" aria-hidden="true" />
      </button>
    </span>
  );
}

type OptionRowProps = {
  label: string;
  checked: boolean;
  count?: number;
  swatch?: string;
  onToggle: () => void;
};

export function OptionRow({ label, checked, count, swatch, onToggle }: OptionRowProps) {
  const id = useId();
  return (
    <label
      htmlFor={id}
      className="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-gray-12 text-sm transition-colors hover:bg-gray-3 has-[:focus-visible]:bg-gray-3"
    >
      <Checkbox id={id} data-row="" checked={checked} onCheckedChange={onToggle} />
      {swatch && (
        <span
          className="size-2 shrink-0 rounded-xs"
          style={{ backgroundColor: swatch }}
          aria-hidden="true"
        />
      )}
      <span className="min-w-0 flex-1 truncate">{label}</span>
      {count !== undefined && (
        <span className="font-mono text-gray-10 text-xs tabular-nums">{formatCount(count)}</span>
      )}
    </label>
  );
}

function moveFocus(container: HTMLElement | null, delta: number): void {
  if (!container) {
    return;
  }
  const rows = [...container.querySelectorAll<HTMLElement>("[data-row]")];
  if (rows.length === 0) {
    return;
  }
  const current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
  const index = current ? rows.indexOf(current) : -1;
  const next = rows[(index + delta + rows.length) % rows.length];
  next.focus();
}

export function onListKeyDown(event: React.KeyboardEvent<HTMLElement>): void {
  if (event.key !== "ArrowDown" && event.key !== "ArrowUp") {
    return;
  }
  event.preventDefault();
  const active = document.activeElement;
  const list = active instanceof HTMLElement ? active.closest<HTMLElement>("[data-list]") : null;
  moveFocus(list ?? event.currentTarget, event.key === "ArrowDown" ? 1 : -1);
}
