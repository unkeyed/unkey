"use client";
import * as PopoverPrimitive from "@radix-ui/react-popover";
import type { PopoverContentProps } from "@radix-ui/react-popover";
import { TriangleWarning2 } from "@unkey/icons";
import React from "react";
import { cn } from "../../lib/utils";
import { Button } from "../buttons/button";
import { Popover, PopoverContent } from "./popover";

const PopoverAnchor = PopoverPrimitive.Anchor;
const PopoverClose = PopoverPrimitive.Close;

type ConfirmVariant = "warning" | "danger";

type ConfirmPopoverProps = {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
  triggerRef: React.RefObject<HTMLElement | null>;
  title?: string;
  description?: string;
  confirmButtonText?: string;
  cancelButtonText?: string;
  variant?: ConfirmVariant;
  popoverProps?: Partial<PopoverContentProps>;
};

const VARIANT_STYLES = {
  warning: {
    iconBg: "bg-warningA-4",
    iconColor: "text-warning-11",
    buttonColor: "warning" as const,
    icon: TriangleWarning2,
  },
  danger: {
    iconBg: "bg-error-4",
    iconColor: "text-error-11",
    buttonColor: "danger" as const,
    icon: TriangleWarning2,
  },
};

const DEFAULT_POPOVER_PROPS = {
  sideOffset: 5,
  side: "bottom" as const,
  align: "center" as const,
  className:
    "bg-white dark:bg-black flex flex-col items-center justify-center border-grayA-4 overflow-hidden !rounded-[10px] p-0 gap-0 min-w-[344px]",
  onOpenAutoFocus: (e: Event) => e.preventDefault(),
};

export const ConfirmPopover = ({
  isOpen,
  onOpenChange,
  onConfirm,
  triggerRef,
  title = "Confirm action",
  description = "Are you sure you want to perform this action?",
  confirmButtonText = "Confirm",
  cancelButtonText = "Cancel",
  variant = "warning",
  popoverProps = {},
}: ConfirmPopoverProps): React.ReactElement => {
  const handleConfirm = () => {
    onConfirm();
    onOpenChange(false);
  };

  const { iconBg, iconColor, buttonColor, icon: Icon } = VARIANT_STYLES[variant];

  // Merge default props with user-provided props, with user props taking precedence
  const mergedPopoverProps = {
    ...DEFAULT_POPOVER_PROPS,
    ...popoverProps,
    // Special handling for className to allow combining classes
    className: cn(DEFAULT_POPOVER_PROPS.className, popoverProps.className),
  };

  // Create a safe ref that Radix can use (virtualRef expects non-null current)
  const safeRef = React.useMemo(
    () => ({
      get current() {
        return triggerRef.current ?? document.body;
      },
    }),
    [triggerRef],
  );

  return (
    <Popover open={isOpen} onOpenChange={onOpenChange}>
      <PopoverAnchor virtualRef={safeRef} />
      <PopoverContent {...mergedPopoverProps}>
        <div className="p-4 w-full">
          <div className="flex gap-3 items-center justify-start">
            <div
              className={cn(
                "flex items-center justify-center rounded size-[22px]",
                iconBg,
                iconColor,
              )}
            >
              <Icon iconSize="sm-regular" />
            </div>
            <div className="font-medium text-[13px] leading-7 text-gray-12">{title}</div>
          </div>
        </div>
        <div className="w-full">
          <div className="h-[1px] bg-grayA-3 w-full" />
        </div>
        <div className="px-4 w-full text-gray-11 text-[13px] leading-6 my-4">{description}</div>
        <div className="space-x-3 w-full px-4 pb-4">
          <Button color={buttonColor} onClick={handleConfirm} className="px-4">
            {confirmButtonText}
          </Button>
          <PopoverClose asChild>
            <Button variant="ghost" className="text-gray-9 px-4">
              {cancelButtonText}
            </Button>
          </PopoverClose>
        </div>
      </PopoverContent>
    </Popover>
  );
};
