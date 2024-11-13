"use client";

import { cn } from "@/lib/utils";
import { Copy, CopyCheck } from "lucide-react";
import { useEffect, useState } from "react";

interface CopyButtonProps extends React.HTMLAttributes<HTMLButtonElement> {
  value: string;
  src?: string;
}

async function copyToClipboardWithMeta(value: string, _meta?: Record<string, unknown>) {
  navigator.clipboard.writeText(value);
}

export function CopyButton({ value, className, src, children, ...props }: CopyButtonProps) {
  const [copied, setCopied] = useState(false);

  useEffect(() => {
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
      aria-label="Copy code snippet"
      type="button"
      className={cn(
        "relative p-1 text-primary focus:outline-none flex items-center gap-2",
        className,
      )}
      onClick={() => {
        copyToClipboardWithMeta(value, {
          component: src,
        });
        setCopied(true);
      }}
      {...props}
    >
      <span className="sr-only">Copy</span>
      {copied ? (
        <CopyCheck className="w-4 h-4 text-white/40" />
      ) : (
        <Copy className="w-4 h-4 text-white/40" />
      )}
      {children}
    </button>
  );
}
