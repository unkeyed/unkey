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
      className={cn("relative p-1 focus:outline-none h-6 w-6 ", className)}
      onClick={() => {
        copyToClipboardWithMeta(value, {
          component: src,
        });
        setCopied(true);
      }}
      {...props}
    >
      <span className="sr-only">Copy</span>
      {copied ? <CopyCheck className="w-full h-full" /> : <Copy className="w-full h-full" />}
    </button>
  );
}
