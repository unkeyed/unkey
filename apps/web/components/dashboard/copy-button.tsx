"use client";

import * as React from "react";

import { cn } from "@/lib/utils";
import { Copy, CopyCheck } from "lucide-react";

interface CopyButtonProps extends React.HTMLAttributes<HTMLButtonElement> {
  value: string;
  src?: string;
}

async function copyToClipboardWithMeta(value: string, _meta?: Record<string, unknown>) {
  navigator.clipboard.writeText(value);
}

export function CopyButton({ value, className, src, ...props }: CopyButtonProps) {
  const timeOutRef = React.useRef<NodeJS.Timeout | null>(null);
  const [hasCopied, setHasCopied] = React.useState(false);

  React.useEffect(() => {
    const clearVisibilityTimeout = () => {
      if (timeOutRef.current) {
        clearTimeout(timeOutRef.current);
        timeOutRef.current = null;
      }
    };

    clearVisibilityTimeout();

    if (hasCopied) {
      const timeOutId = setTimeout(() => {
        setHasCopied(false);
      }, 2000);

      timeOutRef.current = timeOutId;
    }

    return () => {
      clearVisibilityTimeout();
    };
  }, [hasCopied]);

  return (
    <button
      type="button"
      className={cn(
        "relative z-20 h-8 inline-flex items-center justify-center rounded-md border-border p-1 text-sm font-medium text-primary transition-all hover:border-primary hover:bg-secondary focus:outline-none ",
        className,
      )}
      onClick={() => {
        copyToClipboardWithMeta(value, {
          component: src,
        });
        setHasCopied(true);
      }}
      {...props}
    >
      <span className="sr-only">Copy</span>
      {hasCopied ? <CopyCheck className="w-4 h-4" /> : <Copy className="w-4 h-4" />}
    </button>
  );
}
