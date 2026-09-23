"use client";

import { IconChevronDownOutline18 } from "@unkey/icons";
import { Badge } from "@unkey/ui";
import { cn } from "cn";
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
        "bg-gray-1 border rounded-lg",
        "text-[13px] text-gray-12 font-normal",
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
          <Badge variant="count" className="ml-1.5">
            {count}
          </Badge>
        )}
      </span>
      <IconChevronDownOutline18 className="size-3.5 ml-auto shrink-0" />
    </button>
  ),
);
FilterTriggerButton.displayName = "FilterTriggerButton";
