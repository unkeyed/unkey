"use client";

/**
 * Shared display primitives for URNs and permissions: a segment-colored
 * monospace rendering so the grammar is scannable at a glance. Concept pages
 * compose these instead of reinventing permission chips.
 */

import { cn } from "@unkey/ui";
import { parsePermission } from "../lib/urn";

export function UrnText({ value, className }: { value: string; className?: string }) {
  const parsed = parsePermission(value);
  if (!parsed.ok) {
    return <span className={cn("font-mono text-xs text-gray-12", className)}>{value}</span>;
  }
  const { urn, action } = parsed.value;
  const segments = urn.resource.split("/");
  return (
    <span className={cn("font-mono text-xs whitespace-nowrap", className)}>
      <span className="text-gray-9">unkey:v1:</span>
      <span className="text-gray-10">{urn.workspaceID}</span>
      <span className="text-gray-9">:</span>
      {segments.map((segment, i) => (
        <span key={`${i}-${segment}`}>
          {i > 0 && <span className="text-gray-8">/</span>}
          <span
            className={cn(
              segment === "*" || segment === "**"
                ? "text-warning-11 font-semibold"
                : i % 2 === 0
                  ? "text-gray-11"
                  : "text-gray-12",
            )}
          >
            {segment}
          </span>
        </span>
      ))}
      <span className="text-gray-8">#</span>
      <span className={cn(action === "*" ? "text-error-11 font-semibold" : "text-info-11")}>
        {action}
      </span>
    </span>
  );
}

export function PermissionChip({
  value,
  onRemove,
  legacy,
  className,
}: {
  value: string;
  onRemove?: () => void;
  legacy?: boolean;
  className?: string;
}) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-md border border-grayA-4 bg-grayA-2 px-2 py-1",
        legacy && "border-dashed opacity-70",
        className,
      )}
    >
      {legacy ? (
        <span className="font-mono text-xs text-gray-11">{value}</span>
      ) : (
        <UrnText value={value} />
      )}
      {legacy && (
        <span className="rounded bg-warningA-3 px-1 text-[10px] uppercase tracking-wide text-warning-11">
          legacy
        </span>
      )}
      {onRemove && (
        <button
          type="button"
          onClick={onRemove}
          aria-label={`Remove ${value}`}
          className="text-gray-9 hover:text-error-11 transition-colors leading-none"
        >
          &times;
        </button>
      )}
    </span>
  );
}
