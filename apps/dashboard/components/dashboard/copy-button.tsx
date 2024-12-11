"use client";

import type * as React from "react";

import { cn } from "@/lib/utils";
import { useCopyToClipboard } from "@unkey/ui";
import { Copy, CopyCheck } from "lucide-react";

interface CopyButtonProps extends React.HTMLAttributes<HTMLButtonElement> {
  value: string;
  src?: string;
}

export function CopyButton({ value, className, src, ...props }: CopyButtonProps) {
  const [copied, copyToClipboard] = useCopyToClipboard(2000);

  return (
    <button
      type="button"
      className={cn("relative p-1 focus:outline-none h-6 w-6 ", className)}
      onClick={() => copyToClipboard(value)}
      {...props}
    >
      <span className="sr-only">Copy</span>
      {copied ? <CopyCheck className="w-full h-full" /> : <Copy className="w-full h-full" />}
    </button>
  );
}
