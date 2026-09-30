"use client";

import { IconPlusOutline18 } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import type { Route } from "next";
import Link from "next/link";
import type { ReactNode } from "react";

export function CanvasGroup({
  icon,
  label,
  href,
  className,
  children,
}: {
  icon: ReactNode;
  label: string;
  href?: Route;
  className?: string;
  children: ReactNode;
}) {
  return (
    <div
      className={cn(
        "flex min-w-0 max-w-[340px] flex-1 flex-col gap-2 rounded-xl border border-grayA-3 bg-grayA-2 p-2.5",
        className,
      )}
    >
      <div className="flex shrink-0 items-center justify-between px-1 pb-0.5">
        <span className="flex items-center gap-1.5 text-xs text-gray-11 [&_svg]:size-3.5">
          {icon}
          {label}
        </span>
        {href && (
          <Link href={href} className="text-xs text-gray-9 hover:text-gray-12">
            View all
          </Link>
        )}
      </div>
      <div className="flex flex-col gap-2">{children}</div>
    </div>
  );
}

export function CanvasConnector({ dashed = false }: { dashed?: boolean }) {
  return (
    <div
      className={cn(
        "mt-[58px] w-8 shrink-0 border-t",
        dashed ? "border-dashed border-grayA-6" : "border-grayA-7",
      )}
    />
  );
}

export type Tone = "default" | "error" | "warning";

const TONE: Record<Tone, string> = {
  default:
    "bg-raised [--divider:var(--hairline)] hover:border-gray-10 hover:shadow-[0_0_0_3px_var(--color-grayA-3)]",
  error:
    "border-error-6 bg-error-2 [--divider:var(--color-error-6)] hover:border-error-9 hover:shadow-[0_0_0_3px_var(--color-errorA-3)]",
  warning:
    "border-warning-6 bg-warning-2 [--divider:var(--color-warning-6)] hover:border-warning-9 hover:shadow-[0_0_0_3px_var(--color-warningA-3)]",
};

export const TONE_TEXT: Record<Tone, string> = {
  default: "text-gray-11",
  error: "text-error-11",
  warning: "text-warning-11",
};

export function CanvasCard({
  children,
  link,
  tone = "default",
}: {
  children: ReactNode;
  link?: { href: Route; label: string };
  tone?: Tone;
}) {
  return (
    <div
      className={cn(
        "relative rounded-lg border shadow-xs transition-[border-color,box-shadow]",
        TONE[tone],
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

export function CanvasCardHeader({
  icon,
  title,
  mono = false,
  right,
}: {
  icon: ReactNode;
  title: string;
  mono?: boolean;
  right?: ReactNode;
}) {
  return (
    <div className="flex items-center gap-2 px-3 py-2 leading-5 not-last:border-b [border-color:var(--divider)]">
      <span className="text-gray-11 [&_svg]:size-4">{icon}</span>
      <span
        className={cn(
          "min-w-0 truncate text-sm font-medium text-gray-12",
          mono && "font-mono tracking-tight",
        )}
      >
        {title}
      </span>
      {right && <span className="ml-auto shrink-0">{right}</span>}
    </div>
  );
}

export function DetailList({ children }: { children: ReactNode }) {
  return <dl className="flex flex-col px-3 py-1.5">{children}</dl>;
}

export function DetailRow({
  icon,
  label,
  value,
  hint,
  children,
}: {
  icon: ReactNode;
  label: string;
  value: ReactNode;
  hint?: ReactNode;
  children?: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-0.5 py-1 text-sm">
      <div className="flex w-full items-center gap-4">
        <dt className="flex shrink-0 items-center gap-2 font-medium text-gray-11">
          <span className="inline-flex size-4 items-center justify-center text-gray-9 [&_svg]:size-3.5">
            {icon}
          </span>
          {label}
        </dt>
        <dd className="flex min-w-0 flex-1 items-center justify-end gap-1.5 text-right tabular-nums">
          <span className="min-w-0 truncate text-gray-12">{value}</span>
          {hint != null && <span className="shrink-0 text-gray-9">{hint}</span>}
        </dd>
      </div>
      {children && <dd className="flex min-w-0 flex-col pl-6">{children}</dd>}
    </div>
  );
}

const METRIC_COLS = "grid grid-cols-[minmax(0,1fr)_52px_72px] items-center gap-x-3";

export function MetricHeader({
  icon,
  title,
  href,
  columns,
  cols = METRIC_COLS,
}: {
  icon: ReactNode;
  title: string;
  href: Route;
  columns: string[];
  cols?: string;
}) {
  return (
    <div className={cn(cols, "px-3 py-2.5 not-last:border-b [border-color:var(--divider)]")}>
      <Link href={href} className="flex min-w-0 items-center gap-2 hover:text-gray-12">
        <span className="text-gray-11 [&_svg]:size-4">{icon}</span>
        <span className="truncate text-sm font-medium text-gray-12">{title}</span>
      </Link>
      {columns.map((c) => (
        <span key={c} className="text-right text-2xs text-gray-9">
          {c}
        </span>
      ))}
    </div>
  );
}

export function MetricRow({
  href,
  name,
  values,
  cols = METRIC_COLS,
}: {
  href?: Route;
  name: string;
  values: string[];
  cols?: string;
}) {
  const cls = cn(cols, "px-3 py-1.5 text-xs hover:bg-grayA-2");
  const content = (
    <>
      <span className="truncate font-medium text-gray-12">{name}</span>
      {values.map((v, i) => (
        <span
          // biome-ignore lint/suspicious/noArrayIndexKey: columns are positional
          key={i}
          className="truncate text-right text-gray-11 tabular-nums"
        >
          {v}
        </span>
      ))}
    </>
  );
  return href ? (
    <Link href={href} className={cls}>
      {content}
    </Link>
  ) : (
    <div className={cls}>{content}</div>
  );
}

export function GhostCard({
  icon,
  title,
  description,
  onClick,
}: {
  icon: ReactNode;
  title: string;
  description: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="group flex items-start gap-2.5 rounded-lg border bg-raised px-3 py-3 text-left shadow-xs transition-[border-color,box-shadow] hover:border-gray-10 hover:shadow-[0_0_0_3px_var(--color-grayA-3)]"
    >
      <span className="mt-0.5 text-gray-9 group-hover:text-gray-12 [&_svg]:size-4">{icon}</span>
      <span className="min-w-0">
        <span className="block text-sm font-medium text-gray-12">{title}</span>
        <span className="block text-xs text-gray-9">{description}</span>
      </span>
      <IconPlusOutline18 className="ml-auto mt-0.5 size-3.5 text-gray-9 group-hover:text-gray-12" />
    </button>
  );
}
