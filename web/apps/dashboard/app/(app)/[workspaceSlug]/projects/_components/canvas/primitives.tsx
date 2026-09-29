"use client";

import { IconPlusOutline18 } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import type { Route } from "next";
import Link from "next/link";
import { type ReactNode, type RefObject, useLayoutEffect, useRef, useState } from "react";

export function CanvasGroup({
  icon,
  label,
  href,
  fit = "scroll",
  className,
  children,
}: {
  icon: ReactNode;
  label: string;
  href?: Route;
  fit?: "scroll" | "content";
  className?: string;
  children: ReactNode;
}) {
  return (
    <div
      className={cn(
        "flex min-w-0 max-w-[340px] flex-1 flex-col gap-2 rounded-xl border border-grayA-3 bg-grayA-2 p-2.5",
        fit === "scroll" && "max-h-full",
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
      {fit === "scroll" ? (
        <GroupScroll>{children}</GroupScroll>
      ) : (
        <div className="flex flex-col gap-2">{children}</div>
      )}
    </div>
  );
}

function GroupScroll({ children }: { children: ReactNode }) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const more = useMoreBelow(scrollRef);
  return (
    <div
      ref={scrollRef}
      data-wire-scroll
      className={cn(
        "-mx-2.5 -my-1 flex min-h-0 flex-col gap-2 overflow-y-auto px-2.5 py-1 [scrollbar-color:transparent_transparent] [scrollbar-width:thin] hover:[scrollbar-color:var(--color-grayA-6)_transparent]",
        more && "[mask-image:linear-gradient(to_bottom,black_calc(100%-56px),transparent)]",
      )}
    >
      {children}
    </div>
  );
}

function useMoreBelow(ref: RefObject<HTMLDivElement | null>) {
  const [more, setMore] = useState(false);
  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) {
      return;
    }
    const check = () => setMore(el.scrollTop + el.clientHeight < el.scrollHeight - 4);
    check();
    const observer = new ResizeObserver(check);
    observer.observe(el);
    el.addEventListener("scroll", check);
    return () => {
      observer.disconnect();
      el.removeEventListener("scroll", check);
    };
  }, [ref]);
  return more;
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
    "border-border bg-raised [--divider:var(--hairline)] hover:border-gray-10 hover:shadow-[0_0_0_3px_var(--color-grayA-3)] data-[active=true]:border-gray-10 data-[active=true]:shadow-[0_0_0_3px_var(--color-grayA-3)]",
  error:
    "border-error-6 bg-error-2 [--divider:var(--color-error-6)] hover:border-error-9 hover:shadow-[0_0_0_3px_var(--color-errorA-3)] data-[active=true]:border-error-9 data-[active=true]:shadow-[0_0_0_3px_var(--color-errorA-3)]",
  warning:
    "border-warning-6 bg-warning-2 [--divider:var(--color-warning-6)] hover:border-warning-9 hover:shadow-[0_0_0_3px_var(--color-warningA-3)] data-[active=true]:border-warning-9 data-[active=true]:shadow-[0_0_0_3px_var(--color-warningA-3)]",
};

export const TONE_TEXT: Record<Tone, string> = {
  default: "text-gray-11",
  error: "text-error-11",
  warning: "text-warning-11",
};

export function CanvasCard({
  children,
  href,
  external = false,
  tone = "default",
  active = false,
  ...rest
}: {
  children: ReactNode;
  href?: Route | string;
  external?: boolean;
  tone?: Tone;
  active?: boolean;
  onMouseEnter?: () => void;
  onMouseLeave?: () => void;
  "data-wire-app"?: string;
}) {
  const cls = cn(
    "block rounded-lg border shadow-xs transition-[border-color,box-shadow]",
    TONE[tone],
  );
  if (href && external) {
    return (
      <a
        href={href}
        target="_blank"
        rel="noopener noreferrer"
        className={cls}
        data-active={active}
        {...rest}
      >
        {children}
      </a>
    );
  }
  return href ? (
    <Link href={href as Route} className={cls} data-active={active} {...rest}>
      {children}
    </Link>
  ) : (
    <div className={cls} data-active={active} {...rest}>
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
          "min-w-0 truncate text-[13px] font-medium text-gray-12",
          mono && "font-mono tracking-tight",
        )}
      >
        {title}
      </span>
      {right && <span className="ml-auto shrink-0">{right}</span>}
    </div>
  );
}

export const METRIC_COLS = "grid grid-cols-[minmax(0,1fr)_52px_72px] items-center gap-x-3";

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
        <span className="truncate text-[13px] font-medium text-gray-12">{title}</span>
      </Link>
      {columns.map((c) => (
        <span key={c} className="text-right text-[11px] text-gray-9">
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
  active = false,
  wireId,
  onMouseEnter,
  onMouseLeave,
}: {
  href?: Route;
  name: string;
  values: string[];
  cols?: string;
  active?: boolean;
  wireId?: string;
  onMouseEnter?: () => void;
  onMouseLeave?: () => void;
}) {
  const cls = cn(
    cols,
    "px-3 py-1.5 text-xs hover:bg-grayA-2",
    active && "bg-grayA-3 hover:bg-grayA-3",
  );
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
  const props = { "data-wire-ks": wireId, onMouseEnter, onMouseLeave, className: cls };
  return href ? (
    <Link href={href} {...props}>
      {content}
    </Link>
  ) : (
    <div {...props}>{content}</div>
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
      className="group flex items-start gap-2.5 rounded-lg border border-border bg-raised px-3 py-3 text-left shadow-xs transition-[border-color,box-shadow] hover:border-gray-10 hover:shadow-[0_0_0_3px_var(--color-grayA-3)]"
    >
      <span className="mt-0.5 text-gray-9 group-hover:text-gray-12 [&_svg]:size-4">{icon}</span>
      <span className="min-w-0">
        <span className="block text-[13px] font-medium text-gray-12">{title}</span>
        <span className="block text-xs text-gray-9">{description}</span>
      </span>
      <IconPlusOutline18 className="ml-auto mt-0.5 size-3.5 text-gray-9 group-hover:text-gray-12" />
    </button>
  );
}
