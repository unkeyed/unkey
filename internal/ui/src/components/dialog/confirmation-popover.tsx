"use client";
import { Popover, PopoverContent } from "../popover";
import * as PopoverPrimitive from "@radix-ui/react-popover";
import type { PopoverContentProps } from "@radix-ui/react-popover";
import { TriangleWarning2 } from "@unkey/icons";
import { Button } from "../buttons/button";
import { cn } from "../../lib/utils";
import React from "react";

const PopoverAnchor = PopoverPrimitive.Anchor;
const PopoverClose = PopoverPrimitive.Close;

type ConfirmVariant = "warning" | "danger";

type ConfirmPopoverProps = {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
  triggerRef?: React.RefObject<HTMLElement>;
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
  onOpenAutoFocus: (e: Event) => e.stopPropagation(),
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
}: ConfirmPopoverProps): JSX.Element => {
  const defaultRef = React.useRef<HTMLButtonElement>(null);
  const anchorRef = triggerRef || defaultRef;

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

  return (
    <Popover open={isOpen} onOpenChange={onOpenChange}>
      <PopoverAnchor virtualRef={anchorRef} />
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
