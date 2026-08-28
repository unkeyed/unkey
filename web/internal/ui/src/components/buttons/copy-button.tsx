"use client";
import { TaskChecked, TaskUnchecked } from "@unkey/icons";
import * as React from "react";
import { cn } from "../../lib/utils";
import { toast } from "../toaster";
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

async function copyToClipboardWithMeta(value: string, _meta?: Record<string, unknown>) {
  await navigator.clipboard.writeText(value);
}

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
  const [copied, setCopied] = React.useState(false);

  React.useEffect(() => {
    if (!copied) {
      return;
    }
    const timer = setTimeout(() => {
      setCopied(false);
    }, 2000);
    return () => clearTimeout(timer);
  }, [copied]);

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
          try {
            await copyToClipboardWithMeta(value, {
              component: src,
            });
            toast.success("Copied to clipboard", {
              description: toastMessage,
            });
            setCopied(true);
          } catch (e) {
            toast.error("Failed to copy to clipboard", {
              description: e instanceof Error ? e.message : "Unknown error",
            });
          }
          // Call the onClick prop if provided
          onClick?.(e);
        }
      }}
      aria-label="Copy to clipboard"
    >
      <span className="sr-only">Copy</span>
      {copied ? <TaskChecked /> : <TaskUnchecked />}
    </Button>
  );
}
