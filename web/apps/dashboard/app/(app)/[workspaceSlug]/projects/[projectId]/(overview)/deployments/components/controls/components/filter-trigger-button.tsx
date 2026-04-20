"use client";

import { cn } from "@/lib/utils";
import { ChevronDown } from "@unkey/icons";
import { type ButtonHTMLAttributes, type ReactNode, forwardRef } from "react";

type Props = ButtonHTMLAttributes<HTMLButtonElement> & {
  icon?: ReactNode;
  label: ReactNode;
  count?: number;
  isActive?: boolean;
};

export const FilterTriggerButton = forwardRef<HTMLButtonElement, Props>(
  ({ icon, label, count, isActive, disabled, className, ...rest }, ref) => (
    <button
      ref={ref}
      type="button"
      disabled={disabled}
      className={cn(
        "flex items-center gap-2 h-9 px-3 w-full",
        "bg-gray-1 border border-grayA-4 rounded-lg",
        "text-[13px] text-accent-12 font-normal",
        "hover:bg-gray-2 transition-colors",
        isActive && "bg-gray-2",
        disabled && "opacity-50",
        className,
      )}
      {...rest}
    >
      {icon}
      <span className="truncate">
        {label}
        {count !== undefined && count > 0 && (
          <span className="ml-1.5 inline-flex items-center justify-center bg-gray-7 rounded-sm h-4 px-1 text-[11px] font-medium">
            {count}
          </span>
        )}
      </span>
      <ChevronDown className="ml-auto shrink-0" iconSize="md-medium" />
    </button>
  ),
);
FilterTriggerButton.displayName = "FilterTriggerButton";
