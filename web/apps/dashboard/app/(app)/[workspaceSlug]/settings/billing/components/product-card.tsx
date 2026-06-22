"use client";

import { cn } from "@/lib/utils";
import type { ReactNode } from "react";

type ProductCardProps = {
  /** Product icon, rendered inside the colored chip. */
  icon: ReactNode;
  /** Tailwind classes for the icon chip, e.g. "bg-orange-3 text-orange-11". */
  iconClassName: string;
  name: string;
  /** Small tag next to the name, e.g. the plan name or "Add-on". */
  tag?: string;
  /** One-line subtitle under the name: plan fee, included usage, etc. */
  subtitle: ReactNode;
  /** Primary action, top-right: change plan / upgrade / choose a plan. */
  action?: ReactNode;
  children?: ReactNode;
  /** Quiet footer row, right-aligned: where the cancel link lives. */
  footer?: ReactNode;
};

/**
 * Shared shell for the per-product billing cards. Each product gets its own
 * accent color on the icon chip (and its usage meter), which is the only
 * decoration: flat card, one border, no zones.
 */
export const ProductCard: React.FC<ProductCardProps> = ({
  icon,
  iconClassName,
  name,
  tag,
  subtitle,
  action,
  children,
  footer,
}) => {
  return (
    <div className="w-full overflow-hidden rounded-xl border border-grayA-4 bg-white dark:bg-black">
      <div className="flex items-center justify-between gap-4 px-5 py-4">
        <div className="flex min-w-0 items-center gap-3">
          <div
            className={cn(
              "flex size-8 shrink-0 items-center justify-center rounded-md",
              iconClassName,
            )}
          >
            {icon}
          </div>
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <span className="font-medium text-gray-12 text-sm">{name}</span>
              {tag ? (
                <span className="rounded-full bg-grayA-3 px-2 py-0.5 font-medium text-[11px] text-gray-11">
                  {tag}
                </span>
              ) : null}
            </div>
            <div className="truncate text-[13px] text-gray-10">{subtitle}</div>
          </div>
        </div>
        {action ? <div className="shrink-0">{action}</div> : null}
      </div>
      {children ? <div className="border-t border-grayA-3 px-5 py-4">{children}</div> : null}
      {footer ? (
        <div className="flex justify-end border-t border-grayA-3 px-5 py-2.5">{footer}</div>
      ) : null}
    </div>
  );
};
