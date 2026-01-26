"use client";
import { TaskChecked, TaskUnchecked } from "@unkey/icons";
import { toast } from "@unkey/ui";
import { useEffect, useRef, useState } from "react";

type CopyableIDButtonProps = {
  value: string;
  className?: string;
};

async function copyToClipboardWithMeta(value: string, _meta?: Record<string, unknown>) {
  navigator.clipboard.writeText(value);
}

export const CopyableIDButton = ({ value, className = "" }: CopyableIDButtonProps) => {
  const textRef = useRef<HTMLDivElement>(null);
  const pressTimer = useRef<NodeJS.Timeout | null>(null);
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

  const handleMouseDown = () => {
    // Start a long-press timer
    pressTimer.current = setTimeout(() => {
      // For long-press, select the text
      if (textRef.current) {
        const range = document.createRange();
        range.selectNodeContents(textRef.current);
        const selection = window.getSelection();
        if (selection) {
          selection.removeAllRanges();
          selection.addRange(range);
        }
      }
    }, 500);
  };

  const handleMouseUp = () => {
    // Clear the timer if mouse is released before long-press threshold
    if (pressTimer.current) {
      clearTimeout(pressTimer.current);
      pressTimer.current = null;
    }
  };

  const handleMouseLeave = () => {
    // Clear the timer if mouse leaves the button
    if (pressTimer.current) {
      clearTimeout(pressTimer.current);
      pressTimer.current = null;
    }
  };

  const handleClick = (e: React.MouseEvent) => {
    // Only handle click if it wasn't a long press
    if (window.getSelection()?.toString()) {
      // If text is selected, don't trigger the copy
      e.stopPropagation();
    } else {
      // Copy to clipboard
      try {
        copyToClipboardWithMeta(value, {
          component: "CopyableIDButton",
        });
        toast.success("Copied to clipboard", {
          description: value,
        });
        setCopied(true);
      } catch (error) {
        toast.error("Failed to copy to clipboard", {
          description: error instanceof Error ? error.message : "Unknown error",
        });
      }
    }
  };

  return (
    <button
      type="button"
      className={`inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-xs font-mono font-medium external-id h-7 bg-grayA-2 hover:bg-grayA-3 w-[190px] border border-grayA-6 transition-colors focus:ring-0 focus:border-grayA-6 ${className}`}
      onMouseDown={handleMouseDown}
      onMouseUp={handleMouseUp}
      onMouseLeave={handleMouseLeave}
      onClick={handleClick}
      aria-label="Copy ID to clipboard"
      title="Copy to clipboard"
    >
      <div className="flex gap-2 items-center justify-between w-full min-w-0 px-2">
        <div ref={textRef} className="select-text truncate min-w-0 flex-1">
          {value}
        </div>
        <span className="pointer-events-none flex-shrink-0 w-full ">
          {copied ? <TaskChecked className="w-full" /> : <TaskUnchecked className="w-full" />}
        </span>
      </div>
    </button>
  );
};
