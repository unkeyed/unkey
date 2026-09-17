"use client";

import { formatCompactQuantity } from "@/lib/fmt";
import { cn } from "@/lib/utils";
import {
  IconArrowRightOutline18,
  IconChevronRightOutline12,
  IconCubeOutline18,
  IconFingerprintOutline18,
  IconGaugeOutline12,
  IconKey2Outline12,
} from "@unkey/icons";
import type { Route } from "next";
import Link from "next/link";
import type { ReactNode } from "react";
import type { Density, LaunchpadKind, LaunchpadRow, SparkMode } from "./types";

export const ROW_HEIGHT: Record<Density, string> = {
  compact: "h-7",
  default: "h-8",
  roomy: "h-10",
};

export function fmt(value: number): string {
  return value === 0 ? "0" : formatCompactQuantity(value);
}

export function KindGlyph({ kind, className }: { kind: LaunchpadKind; className?: string }) {
  const Glyph = kind === "keyspace" ? IconKey2Outline12 : IconGaugeOutline12;
  return <Glyph className={cn("size-3 shrink-0 text-gray-9", className)} />;
}

export function ProjectChip({ name }: { name: string }) {
  return (
    <span className="inline-flex min-w-0 items-center gap-1 text-gray-9">
      <IconCubeOutline18 className="size-3 shrink-0" />
      <span className="truncate">{name}</span>
    </span>
  );
}

/**
 * Bars are drawn from the row's own max, not a shared scale: the rail ranks by
 * total, so a per-row scale is the only way a quiet resource still shows shape.
 */
export function Spark({
  buckets,
  mode,
  className,
}: {
  buckets: { ok: number; bad: number }[];
  mode: SparkMode;
  className?: string;
}) {
  if (mode === "off" || buckets.length === 0) {
    return null;
  }
  const peak = Math.max(1, ...buckets.map((b) => b.ok + b.bad));
  return (
    <span
      aria-hidden="true"
      className={cn(
        "flex h-4 items-end gap-px",
        mode === "hover" && "opacity-0 transition-opacity group-hover:opacity-100",
        className,
      )}
    >
      {buckets.map((bucket, index) => {
        const value = bucket.ok + bucket.bad;
        const height = Math.max(2, Math.round((value / peak) * 16));
        const bad = value > 0 && bucket.bad / value > 0.1;
        return (
          <span
            // biome-ignore lint/suspicious/noArrayIndexKey: buckets are a fixed-length time series
            key={index}
            className={cn("w-[3px] rounded-[1px]", bad ? "bg-warning-9" : "bg-grayA-7")}
            style={{ height }}
          />
        );
      })}
    </span>
  );
}

export function SectionLabel({
  children,
  action,
  className,
}: {
  children: ReactNode;
  action?: { label: string; href: Route };
  className?: string;
}) {
  return (
    <div className={cn("flex h-8 items-center justify-between gap-2 px-1", className)}>
      <span className="truncate text-[13px] font-medium text-accent-12">{children}</span>
      {action && (
        <Link
          href={action.href}
          className="shrink-0 text-xs text-gray-9 transition-colors hover:text-accent-12"
        >
          {action.label}
        </Link>
      )}
    </div>
  );
}

export function CountPill({ value }: { value: number }) {
  return (
    <span className="rounded-full bg-grayA-3 px-1.5 text-[11px] font-medium tabular-nums text-gray-11">
      {value}
    </span>
  );
}

export function SummaryRow({
  icon,
  label,
  value,
  href,
  density,
}: {
  icon: ReactNode;
  label: string;
  value: string;
  href: Route;
  density: Density;
}) {
  return (
    <Link
      href={href}
      className={cn(
        "group flex items-center gap-2 rounded-md px-2 text-[13px] transition-colors hover:bg-grayA-2",
        ROW_HEIGHT[density],
      )}
    >
      {icon}
      <span className="min-w-0 flex-1 truncate text-gray-11">{label}</span>
      <span className="shrink-0 tabular-nums text-accent-12">{value}</span>
      <IconChevronRightOutline12 className="size-3 shrink-0 text-gray-8" />
    </Link>
  );
}

export function IdentitiesRow({
  count,
  href,
  density,
}: {
  count: number;
  href: Route;
  density: Density;
}) {
  return (
    <SummaryRow
      icon={<IconFingerprintOutline18 className="size-3 shrink-0 text-gray-9" />}
      label="Identities"
      value={fmt(count)}
      href={href}
      density={density}
    />
  );
}

export type Action = { label: string; hint?: string; href: Route };

export function ActionRow({ action, density }: { action: Action; density: Density }) {
  return (
    <Link
      href={action.href}
      className={cn(
        "group flex items-center gap-2 rounded-md px-2 text-[13px] transition-colors hover:bg-grayA-2",
        ROW_HEIGHT[density],
      )}
    >
      <span className="min-w-0 flex-1 truncate text-accent-12">{action.label}</span>
      {action.hint && <span className="shrink-0 text-xs text-gray-9">{action.hint}</span>}
      <IconArrowRightOutline18 className="size-3 shrink-0 text-gray-8 transition-transform group-hover:translate-x-0.5" />
    </Link>
  );
}

export function RowSkeleton({ density, count = 5 }: { density: Density; count?: number }) {
  return (
    <div aria-busy="true" className="flex flex-col">
      {Array.from({ length: count }).map((_, index) => (
        <div
          // biome-ignore lint/suspicious/noArrayIndexKey: skeleton rows need no stable key
          key={index}
          className={cn("flex items-center gap-2 px-2", ROW_HEIGHT[density])}
        >
          <div className="h-2.5 flex-1 rounded bg-grayA-3" />
          <div className="h-2.5 w-10 rounded bg-grayA-3" />
        </div>
      ))}
    </div>
  );
}

export function RowLink({
  row,
  density,
  className,
  children,
}: {
  row: LaunchpadRow;
  density: Density;
  className?: string;
  children: ReactNode;
}) {
  return (
    <Link
      href={row.href}
      className={cn(
        "group flex items-center gap-2 rounded-md px-2 text-[13px] transition-colors hover:bg-grayA-2",
        ROW_HEIGHT[density],
        className,
      )}
    >
      {children}
    </Link>
  );
}
