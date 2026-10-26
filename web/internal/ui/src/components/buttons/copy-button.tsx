"use client";
import { IconCheckOutline18, IconCloneOutline18 } from "@unkey/icons";
import type * as React from "react";
import { useCopyToClipboard } from "../../hooks/use-copy-to-clipboard";
import { cn } from "../../lib/utils";
import { Button, type ButtonProps } from "./button";

type CopyButtonProps = ButtonProps & {
  /**
   * The value to copy to clipboard
   */
  value: string;
  /**
   * Source component for analytics
   */
  src?: string;
  /**
   * toast message to show when copied
   */
  toastMessage?: string;
  ref?: React.Ref<HTMLButtonElement>;
};

export function CopyButton({
  value,
  src,
  variant = "outline",
  className,
  toastMessage,
  onClick,
  ref,
  ...props
}: CopyButtonProps) {
  const { copied, copy } = useCopyToClipboard(value, toastMessage);

  return (
    <Button
      {...props}
      ref={ref}
      type="button"
      variant={variant}
      title="Copy to clipboard"
      size="icon"
      className={cn("focus:ring-0 focus:border-grayA-6 secret p-0", className)}
      onClick={async (e) => {
        if (!e.defaultPrevented) {
          e.stopPropagation(); // Prevent triggering parent button click
          await copy();
          onClick?.(e);
        }
      }}
      aria-label="Copy to clipboard"
    >
      <span className="sr-only">Copy</span>
      {copied ? <IconCheckOutline18 /> : <IconCloneOutline18 />}
    </Button>
  );
}
