"use client";

import * as React from "react";

import { cn } from "@/lib/utils";
import { Check, Copy } from "lucide-react";

interface CopyButtonProps extends React.HTMLAttributes<HTMLButtonElement> {
  value: string;
  src?: string;
}

async function copyToClipboardWithMeta(value: string, _meta?: Record<string, unknown>) {
  navigator.clipboard.writeText(value);
}

export function CopyButton({ value, className, src, ...props }: CopyButtonProps) {
  const [hasCopied, setHasCopied] = React.useState(false);

  React.useEffect(() => {
    setTimeout(() => {
      setHasCopied(false);
    }, 2000);
  }, [hasCopied]);

  return (
    <button
      type="button"
      className={cn(
        "relative z-20 inline-flex h-8 items-center justify-center rounded-md border-gray-200 p-2 text-sm font-medium text-gray-900 transition-all hover:bg-gray-100 focus:outline-none dark:text-gray-100 dark:hover:bg-gray-800",
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
      {hasCopied ? <Check className="w-4 h-4" /> : <Copy className="w-4 h-4" />}
    </button>
  );
}
