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
};

async function copyToClipboardWithMeta(value: string, _meta?: Record<string, unknown>) {
  navigator.clipboard.writeText(value);
}

export const CopyButton = React.forwardRef<HTMLButtonElement, CopyButtonProps>(
  ({ value, src, variant = "outline", className, toastMessage, onClick, ...props }, ref) => {
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
        className={cn("focus:ring-0 focus:border-grayA-6 secret", className)}
        onClick={(e) => {
          if (!e.defaultPrevented) {
            e.stopPropagation(); // Prevent triggering parent button click
            try {
              copyToClipboardWithMeta(value, {
                component: src,
              });
              toast.success("Copied to clipboard", {
                description: toastMessage,
              });
            } catch (e) {
              toast.error("Failed to copy to clipboard", {
                description: e instanceof Error ? e.message : "Unknown error",
              });
            }
            setCopied(true);
            // Call the onClick prop if provided
            onClick?.(e);
          }
        }}
        aria-label="Copy to clipboard"
      >
        <span className="sr-only">Copy</span>
        {copied ? (
          <TaskChecked className="w-full h-full" />
        ) : (
          <TaskUnchecked className="w-full h-full" />
        )}
      </Button>
    );
  },
);

CopyButton.displayName = "CopyButton";
