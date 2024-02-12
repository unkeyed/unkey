"use client";

import { cn } from "@/lib/utils";
import { CopyCheck } from "lucide-react";
import * as React from "react";
import { BlogCodeCopy } from "./svg/blog-code-block";

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
      className={cn("relative p-1 text-primary focus:outline-none ", className)}
      onClick={() => {
        copyToClipboardWithMeta(value, {
          component: src,
        });
        setHasCopied(true);
      }}
      {...props}
    >
      <span className="sr-only">Copy</span>
      {hasCopied ? <CopyCheck className="w-6 h-6" /> : <BlogCodeCopy className="w-6 h-6" />}
    </button>
  );
}
