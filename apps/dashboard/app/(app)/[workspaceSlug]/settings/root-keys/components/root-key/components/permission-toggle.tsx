"use client";
import { cn } from "@/lib/utils";
import type { CheckedState } from "@radix-ui/react-checkbox";
import { ChevronRight } from "@unkey/icons";
import { Checkbox, InfoTooltip } from "@unkey/ui";
import type React from "react";
import { forwardRef, useId } from "react";

type PermissionToggleProps = React.HTMLAttributes<HTMLDivElement> & {
  category: string | React.ReactNode;
  checked: CheckedState;
  setChecked: (checked: boolean) => void;
  label: string | React.ReactNode;
  description: string;
  className?: string;
};

const PermissionToggle = forwardRef<HTMLDivElement, PermissionToggleProps>(
  ({ category, checked, setChecked, label, description, className, ...props }, ref) => {
    const id = useId();

    return (
      <div
        ref={ref}
        className={cn(
          "hover:cursor-pointer flex flex-row items-center justify-start gap-4 transition-all pl-3 h-full mb-1 ml-2 w-full hover:bg-grayA-3 rounded-lg",
          className,
        )}
        {...props}
      >
        <Checkbox
          id={id}
          size="lg"
          checked={checked}
          aria-labelledby={`${id}-label`}
          aria-describedby={`${id}-description`}
          onCheckedChange={(next) => setChecked(next === true)}
          onClick={(e) => e.stopPropagation()}
        />
        <div className="flex flex-col text-left min-w-48 max-w-full gap-1">
          <div className="inline-flex items-center gap-2 w-full">
            <span id={`${id}-label`} className="text-sm w-fit">
              {category}
            </span>
            <ChevronRight iconsize="sm-regular" className="text-grayA-8" />
            {<span className="text-sm w-full">{label}</span>}
          </div>
          <InfoTooltip content={description} className="w-full text-left">
            <p id={`${id}-description`} className="text-xs text-gray-10 text-left w-full truncate">
              {description}
            </p>
          </InfoTooltip>
        </div>
      </div>
    );
  },
);

PermissionToggle.displayName = "PermissionToggle";

export { PermissionToggle };
