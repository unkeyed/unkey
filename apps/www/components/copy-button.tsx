"use client";
import { cn } from "@/lib/utils";
import { useCopyToClipboard } from "@unkey/ui";
import { Copy, CopyCheck } from "lucide-react";

interface CopyButtonProps extends React.HTMLAttributes<HTMLButtonElement> {
  value: string;
  src?: string;
}

export function CopyButton({ value, className, src, children, ...props }: CopyButtonProps) {
  const [copied, copyToClipboard] = useCopyToClipboard(2000);

  return (
    <button
      aria-label="Copy code snippet"
      type="button"
      className={cn(
        "relative p-1 text-primary focus:outline-none flex items-center gap-2",
        className,
      )}
      onClick={() => copyToClipboard(value)}
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
