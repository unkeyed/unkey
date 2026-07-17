"use client";

/**
 * Monospace rendering of a permission template, mirroring UrnText styling but
 * with typed holes drawn as accent pills inline. Filled holes render as normal
 * path segments so a partially filled template reads like a real URN.
 */

import { cn } from "@unkey/ui";
import { WORKSPACE_ID } from "../lib/mock-data";
import { HOLE_META, type HoleKind, type HoleSelections, type PermissionTemplate } from "./recipes";

export function HolePill({ kind, className }: { kind: HoleKind; className?: string }) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded border border-accent-7 bg-accent-3 px-1 font-mono text-[11px] font-medium leading-4 text-accent-12",
        className,
      )}
    >
      {HOLE_META[kind].placeholder}
    </span>
  );
}

function PathFragment({ value, startWithSlash }: { value: string; startWithSlash: boolean }) {
  const segments = value.split("/");
  return (
    <>
      {segments.map((segment, i) => (
        <span key={`${i}-${segment}`}>
          {(i > 0 || startWithSlash) && <span className="text-gray-8">/</span>}
          <span
            className={cn(
              segment === "*" || segment === "**"
                ? "font-semibold text-warning-11"
                : "text-gray-11",
            )}
          >
            {segment}
          </span>
        </span>
      ))}
    </>
  );
}

export function TemplateLine({
  template,
  selections,
  className,
}: {
  template: PermissionTemplate;
  selections?: HoleSelections;
  className?: string;
}) {
  return (
    <span className={cn("whitespace-nowrap font-mono text-xs", className)}>
      <span className="text-gray-9">unkey:v1:</span>
      <span className="text-gray-10">{WORKSPACE_ID}</span>
      <span className="text-gray-9">:</span>
      {template.parts.map((part, i) => {
        if (part.kind === "literal") {
          return (
            // biome-ignore lint/suspicious/noArrayIndexKey: parts are static per template
            <PathFragment key={i} value={part.value} startWithSlash={i > 0} />
          );
        }
        const chosen = selections?.[part.hole];
        if (chosen !== undefined) {
          return (
            // biome-ignore lint/suspicious/noArrayIndexKey: parts are static per template
            <PathFragment key={i} value={chosen} startWithSlash={i > 0} />
          );
        }
        return (
          // biome-ignore lint/suspicious/noArrayIndexKey: parts are static per template
          <span key={i}>
            {i > 0 && <span className="text-gray-8">/</span>}
            <HolePill kind={part.hole} />
          </span>
        );
      })}
      <span className="text-gray-8">#</span>
      <span
        className={cn(template.action === "*" ? "font-semibold text-error-11" : "text-info-11")}
      >
        {template.action}
      </span>
    </span>
  );
}
