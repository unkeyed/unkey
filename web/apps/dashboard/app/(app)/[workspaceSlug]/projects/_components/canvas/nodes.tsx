"use client";

import { IconPlusOutline12 } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import type { Route } from "next";
import Link from "next/link";
import type { ComponentProps, ReactNode } from "react";

export type NodeEdge = "border" | "ring";
export type NodeTone = "default" | "error" | "warning";

const NODE_EDGE: Record<NodeEdge, Record<NodeTone, string>> = {
  border: {
    default:
      "border bg-raised shadow-xs hover:border-gray-10 hover:shadow-[0_0_0_3px_var(--color-grayA-3)]",
    error:
      "border border-error-6 bg-error-2 shadow-xs hover:border-error-9 hover:shadow-[0_0_0_3px_var(--color-errorA-3)]",
    warning:
      "border border-warning-6 bg-warning-2 shadow-xs hover:border-warning-9 hover:shadow-[0_0_0_3px_var(--color-warningA-3)]",
  },
  ring: {
    default:
      "bg-raised shadow-sm ring-1 ring-grayA-5 hover:ring-grayA-8 data-[active=true]:ring-grayA-8",
    error: "bg-raised shadow-sm ring-1 ring-error-7",
    warning: "bg-raised shadow-sm ring-1 ring-warning-7",
  },
};

export function Node({
  edge,
  tone = "default",
  active = false,
  link,
  className,
  children,
  ...rest
}: {
  edge: NodeEdge;
  tone?: NodeTone;
  active?: boolean;
  link?: { href: Route; label: string };
  className?: string;
  children: ReactNode;
} & Pick<ComponentProps<"div">, "onMouseEnter" | "onMouseLeave" | "title">) {
  return (
    <div
      {...rest}
      data-active={active}
      className={cn(
        "relative flex min-w-0 animate-pop flex-col overflow-hidden rounded-lg transition-[border-color,box-shadow] motion-reduce:animate-none",
        NODE_EDGE[edge][tone],
        className,
      )}
    >
      {link && (
        <Link
          href={link.href}
          aria-label={link.label}
          className="absolute inset-0 rounded-lg focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-7"
        />
      )}
      {children}
    </div>
  );
}

type NodeLead = { icon: ReactNode; leading?: never } | { leading: ReactNode; icon?: never };

export function NodeHeader({
  icon,
  leading,
  title,
  meta,
  right,
}: NodeLead & { title: string; meta?: ReactNode; right?: ReactNode }) {
  return (
    <span className="flex min-w-0 items-center gap-1.5 px-3 py-2">
      {icon != null ? (
        <span className="shrink-0 text-gray-11 [&_svg]:size-3.5">{icon}</span>
      ) : (
        leading
      )}
      <span className="shrink-0 text-xs font-medium text-gray-12">{title}</span>
      {meta != null && <span className="truncate font-mono text-[11px] text-gray-10">{meta}</span>}
      {right != null && <span className="ml-auto flex shrink-0 items-center gap-1">{right}</span>}
    </span>
  );
}

export type BadgeTone = "default" | "warning";

const BADGE: Record<BadgeTone, string> = {
  default: "border-grayA-5 bg-grayA-3 font-mono font-medium text-gray-12",
  warning: "border-warning-6 bg-warning-3 text-warning-11",
};

export function Badge({ tone = "default", children }: { tone?: BadgeTone; children: ReactNode }) {
  return (
    <span className={cn("rounded-sm border px-1 text-[10px] leading-4", BADGE[tone])}>
      {children}
    </span>
  );
}

const STATUS_TEXT: Record<NodeTone, string> = {
  default: "text-gray-11",
  error: "text-error-11",
  warning: "text-warning-11",
};

export function StatusBadge({
  dotClass,
  tone = "default",
  children,
}: {
  dotClass: string;
  tone?: NodeTone;
  children: ReactNode;
}) {
  return (
    <span className={cn("inline-flex items-center gap-1 text-[11px]", STATUS_TEXT[tone])}>
      <span className={cn("size-1.5 shrink-0 rounded-full", dotClass)} />
      {children}
    </span>
  );
}

export function Rows({ children }: { children: ReactNode }) {
  return (
    <span className="flex flex-col gap-1 border-t border-grayA-4 px-3 py-2 text-[11px]">
      {children}
    </span>
  );
}

export function RowItem({
  icon,
  label,
  value,
  hint,
}: {
  icon: ReactNode;
  label: string;
  value: ReactNode;
  hint?: ReactNode;
}) {
  return (
    <span className="flex min-w-0 items-center gap-2">
      <span className="shrink-0 text-gray-10 [&_svg]:size-3">{icon}</span>
      <span className="shrink-0 font-medium text-gray-11">{label}</span>
      <span className="ml-auto flex min-w-0 items-center gap-1.5">
        <span className="truncate font-mono text-gray-12">{value}</span>
        {hint != null && <span className="shrink-0 text-gray-10">{hint}</span>}
      </span>
    </span>
  );
}

export function SpecGrid({ children }: { children: ReactNode }) {
  return (
    <span className="grid grid-cols-2 gap-x-3 gap-y-1 whitespace-nowrap border-t border-grayA-4 px-3 py-2 font-mono text-[10px]">
      {children}
    </span>
  );
}

export function SpecItem({
  icon,
  label,
  value,
}: {
  icon: ReactNode;
  label: string;
  value: ReactNode;
}) {
  return (
    <span className="flex min-w-0 items-center gap-1" title={label}>
      <span className="shrink-0 text-gray-10 [&_svg]:size-3" aria-label={label}>
        {icon}
      </span>
      <span className="truncate text-gray-12">{value}</span>
    </span>
  );
}

export function Group({
  icon,
  label,
  children,
}: {
  icon: ReactNode;
  label: string;
  children: ReactNode;
}) {
  return (
    <div className="flex min-w-0 max-w-[320px] flex-1 flex-col gap-2 rounded-xl border border-grayA-3 bg-grayA-2 p-2">
      <span className="flex items-center gap-1.5 px-1 text-[11px] text-gray-11 [&_svg]:size-3.5">
        {icon}
        {label}
      </span>
      <div className="flex flex-col gap-2">{children}</div>
    </div>
  );
}

export function GhostNode({
  icon,
  title,
  description,
  href,
}: {
  icon: ReactNode;
  title: string;
  description: string;
  href: Route;
}) {
  return (
    <Link
      href={href}
      className="group flex items-start gap-2 rounded-lg border border-dashed border-grayA-6 bg-raised px-3 py-2.5 transition-colors hover:border-grayA-8"
    >
      <span className="mt-px text-gray-10 group-hover:text-gray-12 [&_svg]:size-3.5">{icon}</span>
      <span className="flex min-w-0 flex-col">
        <span className="text-xs font-medium text-gray-12">{title}</span>
        <span className="text-[11px] text-gray-10">{description}</span>
      </span>
      <IconPlusOutline12 className="mt-0.5 ml-auto size-3 shrink-0 text-gray-10 group-hover:text-gray-12" />
    </Link>
  );
}

function FlowPath({ d, stroke, empty }: { d: string; stroke: string; empty: boolean }) {
  return (
    <path
      d={d}
      fill="none"
      strokeWidth={1}
      strokeDasharray="3 3"
      vectorEffect="non-scaling-stroke"
      className={
        empty ? "stroke-gray-6" : cn(stroke, "animate-dash-flow motion-reduce:animate-none")
      }
    />
  );
}

export function HFlow({ empty }: { empty: boolean }) {
  return (
    <svg className="mt-[54px] h-2 w-8 shrink-0" viewBox="0 0 40 8" aria-hidden="true">
      <FlowPath d="M0,4 H40" stroke="stroke-gray-8" empty={empty} />
    </svg>
  );
}

export function FanOut({ targets, empty = false }: { targets: number; empty?: boolean }) {
  const n = Math.max(targets, 1);
  return (
    <svg
      viewBox="0 0 100 100"
      preserveAspectRatio="none"
      className="pointer-events-none h-10 w-full"
      aria-hidden="true"
    >
      {Array.from({ length: n }, (_, i) => ((i + 0.5) / n) * 100).map((x) => (
        <FlowPath key={x} d={`M50,0 V50 H${x} V100`} stroke="stroke-gray-7" empty={empty} />
      ))}
    </svg>
  );
}
