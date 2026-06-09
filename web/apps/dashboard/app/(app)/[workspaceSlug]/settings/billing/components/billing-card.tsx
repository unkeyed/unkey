"use client";

import { cn } from "@/lib/utils";
import type { ReactNode } from "react";

/**
 * Local billing-page primitives in the marketing-site visual language:
 * square bordered panels, mono uppercase labels, glowing section dots.
 * Kept local to the billing page so the shared @unkey/ui SettingCard used
 * elsewhere is untouched.
 */

/** Diagonal hatching used as a background texture on passive panels. */
export const billingStripes =
  "bg-[repeating-linear-gradient(135deg,var(--color-grayA-3)_0_1px,transparent_1px_8px)]";

/**
 * Squares off @unkey/ui buttons to sit inside the sharp-cornered panels:
 * no radius, no drop shadow.
 */
export const billingButton = "rounded-none drop-shadow-none";

/** Mono uppercase tag marking a page section. */
export function BillingLabel({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <span
      className={cn(
        "inline-flex w-fit items-center border border-grayA-6 px-3 py-1.5 font-mono text-[11px] text-gray-12 uppercase tracking-[0.08em]",
        className,
      )}
    >
      {children}
    </span>
  );
}

/** A labelled page section: tag on top, content below. */
export function BillingSection({
  label,
  children,
  className,
}: {
  label: string;
  children: ReactNode;
  className?: string;
}) {
  return (
    <section className={cn("flex w-full flex-col gap-3", className)}>
      <BillingLabel>{label}</BillingLabel>
      {children}
    </section>
  );
}

export function BillingCardGroup({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("w-full divide-y divide-grayA-3 border border-grayA-4", className)}>
      {children}
    </div>
  );
}

export function BillingCard({
  label,
  title,
  description,
  children,
  footer,
  className,
}: {
  /** Mono uppercase eyebrow above the title. */
  label?: string;
  title?: ReactNode;
  description?: ReactNode;
  /** Right-aligned value or action. */
  children?: ReactNode;
  /** Full-width content below the row, e.g. a progress bar. */
  footer?: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("flex flex-col gap-4 px-5 py-4", className)}>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex flex-col gap-1">
          {label ? (
            <span className="font-mono text-[11px] text-gray-9 uppercase tracking-wider">
              {label}
            </span>
          ) : null}
          {title ? <div className="font-medium text-gray-12 text-sm">{title}</div> : null}
          {description ? (
            <div className="text-[13px] text-gray-10 leading-snug">{description}</div>
          ) : null}
        </div>
        {children ? (
          <div className="flex shrink-0 items-center gap-3 sm:justify-end">{children}</div>
        ) : null}
      </div>
      {footer ?? null}
    </div>
  );
}
