import { cn } from "@/lib/utils";
import { Clipboard } from "@unkey/icons";
import { useState } from "react";

interface CopyCellProps {
  value: string;
  displayValue?: string;
  className?: string;
  monospace?: boolean;
}

/**
 * Copyable cell with click-to-copy functionality
 */
export function CopyCell({ value, displayValue, className, monospace = false }: CopyCellProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <button
      type="button"
      onClick={handleCopy}
      className={cn(
        "group flex items-center gap-2 text-xs text-accent-11 hover:text-accent-12 transition-colors",
        "focus:outline-none focus:text-accent-12",
        monospace && "font-mono",
        className,
      )}
      title="Click to copy"
    >
      <span className="truncate">{displayValue || value}</span>
      <Clipboard
        className={cn(
          "size-3 flex-shrink-0 opacity-0 group-hover:opacity-100 transition-opacity",
          copied && "opacity-100 text-success-11",
        )}
      />
      {copied && <span className="text-xs text-success-11">Copied!</span>}
    </button>
  );
}
