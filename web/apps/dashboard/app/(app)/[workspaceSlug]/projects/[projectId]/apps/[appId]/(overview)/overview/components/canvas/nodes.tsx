"use client";

import { IconDotsOutline18, IconPlusOutline12 } from "@unkey/icons";
import { CopyButton, InfoTooltip } from "@unkey/ui";
import { cn } from "cn";
import type { Route } from "next";
import Link from "next/link";
import type { ReactNode } from "react";

export type NodeTone = "default" | "error";

const NODE_TONE: Record<NodeTone, { base: string; linkHover: string }> = {
  default: {
    base: "bg-raised shadow-xs ring-1 ring-grayA-4",
    linkHover: "hover:ring-gray-10 hover:outline-2 hover:outline-offset-1 hover:outline-grayA-3",
  },
  error: {
    base: "bg-error-2 shadow-xs ring-1 ring-error-6",
    linkHover: "hover:ring-error-9 hover:outline-2 hover:outline-offset-1 hover:outline-errorA-3",
  },
};

export function Node({
  tone = "default",
  link,
  children,
}: {
  tone?: NodeTone;
  link?: { href: Route; label: string };
  children: ReactNode;
}) {
  return (
    <div
      className={cn(
        "relative flex min-w-0 flex-col overflow-hidden rounded-lg transition",
        NODE_TONE[tone].base,
        link && NODE_TONE[tone].linkHover,
      )}
    >
      {link && (
        <Link
          href={link.href}
          aria-label={link.label}
          className="absolute inset-0 rounded-lg focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-gray-7"
        />
      )}
      {children}
    </div>
  );
}

export function NodeHeader({
  icon,
  name,
  right,
}: {
  icon: ReactNode;
  name: ReactNode;
  right?: ReactNode;
}) {
  return (
    <span className="flex min-w-0 items-center gap-1.5 px-3 py-2">
      <span className="shrink-0 text-gray-11 [&_svg]:size-3.5">{icon}</span>
      <span className="min-w-0 truncate text-xs font-medium text-gray-12">{name}</span>
      {right != null && <span className="ml-auto flex shrink-0 items-center gap-1">{right}</span>}
    </span>
  );
}

const STATUS_TEXT: Record<NodeTone, string> = {
  default: "text-gray-11",
  error: "text-error-11",
};

export function StatusBadge({
  indicator,
  tone = "default",
  children,
}: {
  indicator: ReactNode;
  tone?: NodeTone;
  children: ReactNode;
}) {
  return (
    <span className={cn("inline-flex items-center gap-1 text-2xs", STATUS_TEXT[tone])}>
      {indicator}
      {children}
    </span>
  );
}

export function Rows({ children }: { children: ReactNode }) {
  return (
    <span className="flex flex-col gap-1 border-t border-grayA-4 px-3 py-2 text-2xs">
      {children}
    </span>
  );
}

export function RowItem({
  icon,
  label,
  value,
  fullValue,
  hint,
  copy,
}: {
  icon: ReactNode;
  label: string;
  value: ReactNode;
  fullValue?: string;
  hint?: ReactNode;
  copy?: string;
}) {
  const shown = <span className="truncate font-mono text-gray-12">{value}</span>;
  return (
    <span className="flex min-w-0 items-center gap-2">
      <span className="shrink-0 text-gray-10 [&_svg]:size-3">{icon}</span>
      <span className="shrink-0 font-medium text-gray-11">{label}</span>
      <span className="ml-auto flex min-w-0 items-center gap-1.5">
        {fullValue ? (
          <InfoTooltip asChild content={fullValue} position={{ side: "top" }}>
            <span className="relative z-10 min-w-0 truncate">{shown}</span>
          </InfoTooltip>
        ) : (
          shown
        )}
        {hint != null && <span className="shrink-0 text-gray-10">{hint}</span>}
        {copy && (
          <CopyButton
            value={copy}
            variant="ghost"
            className="relative z-10 size-4 shrink-0 p-0 [&_svg]:size-3"
            toastMessage={copy}
            src="app-canvas"
          />
        )}
      </span>
    </span>
  );
}

export function MoreRow({ count, href }: { count: number; href: Route }) {
  if (count <= 0) {
    return null;
  }
  return (
    <Link
      href={href}
      className="relative z-10 flex items-center gap-2 self-start text-gray-10 hover:text-gray-12 rounded-sm focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-7"
    >
      <span className="shrink-0 [&_svg]:size-3">
        <IconDotsOutline18 />
      </span>
      {count} more
    </Link>
  );
}

export function Canvas({
  label,
  focusable = true,
  children,
}: {
  label: string;
  focusable?: boolean;
  children: ReactNode;
}) {
  return (
    <div className="w-full rounded-t-lg bg-gray-3/40 last:rounded-b-lg">
      <section
        // Focusable so keyboard users can scroll a wide canvas
        tabIndex={focusable ? 0 : undefined}
        aria-label={label}
        className="overflow-x-auto rounded-t-lg focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-gray-7"
      >
        <div className="flex w-full min-w-225 items-start px-5 py-5">{children}</div>
      </section>
    </div>
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
    <div className="flex min-w-0 flex-1 flex-col gap-2 rounded-xl border border-grayA-3 bg-grayA-2 p-2">
      <GroupLabel icon={icon} label={label} />
      <div className="flex flex-col gap-2">{children}</div>
    </div>
  );
}

function GroupLabel({ icon, label }: { icon: ReactNode; label: string }) {
  return (
    <span className="flex items-center gap-1.5 px-1 text-2xs text-gray-11 [&_svg]:size-3.5">
      {icon}
      {label}
    </span>
  );
}

export function GhostNode({
  icon,
  name,
  description,
  href,
}: {
  icon: ReactNode;
  name: string;
  description: string;
  href: Route;
}) {
  return (
    <Link
      href={href}
      className="group flex items-start gap-2 rounded-lg border border-dashed border-grayA-6 bg-raised px-3 py-2.5 transition-colors hover:border-grayA-8 focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-7"
    >
      <span className="mt-px text-gray-10 group-hover:text-gray-12 [&_svg]:size-3.5">{icon}</span>
      <span className="flex min-w-0 flex-col">
        <span className="text-xs font-medium text-gray-12">{name}</span>
        <span className="text-2xs text-gray-10">{description}</span>
      </span>
      <IconPlusOutline12 className="mt-0.5 ml-auto size-3 shrink-0 text-gray-10 group-hover:text-gray-12" />
    </Link>
  );
}

// Mirrors a Group's border, padding and label so the line meets each node header
export function HFlow({ empty }: { empty: boolean }) {
  return (
    <div aria-hidden="true" className="flex shrink-0 flex-col gap-2 border border-transparent py-2">
      <span className="invisible">
        <GroupLabel icon={<IconDotsOutline18 />} label="&nbsp;" />
      </span>
      <span className="flex h-8 items-center">
        <svg className="h-2 w-8" viewBox="0 0 40 8">
          <path
            d="M0,4 H40"
            fill="none"
            strokeWidth={1}
            strokeDasharray="3 3"
            vectorEffect="non-scaling-stroke"
            className={
              empty ? "stroke-gray-6" : "animate-dash-flow stroke-gray-8 motion-reduce:animate-none"
            }
          />
        </svg>
      </span>
    </div>
  );
}
