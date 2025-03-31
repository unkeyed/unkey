"use client";
import { cn } from "@/lib/utils";
import { TaskChecked, TaskUnchecked } from "@unkey/icons";
import * as React from "react";

interface CopyButtonProps extends React.HTMLAttributes<HTMLButtonElement> {
  value: string;
  src?: string;
}

async function copyToClipboardWithMeta(value: string, _meta?: Record<string, unknown>) {
  navigator.clipboard.writeText(value);
}

export const CopyButton = React.forwardRef<HTMLButtonElement, CopyButtonProps>(
  ({ value, className, src, ...props }, ref) => {
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
      <button
        type="button"
        ref={ref}
        className={cn("relative p-1 focus:outline-none h-6 w-6 ", className)}
        onClick={(e) => {
          e.stopPropagation(); // Prevent triggering parent button click
          copyToClipboardWithMeta(value, {
            component: src,
          });
          setCopied(true);
        }}
        {...props}
      >
        <span className="sr-only">Copy</span>
        {copied ? (
          <TaskChecked className="w-full h-full" />
        ) : (
          <TaskUnchecked className="w-full h-full" />
        )}
      </button>
    );
  },
);

CopyButton.displayName = "CopyButton";
