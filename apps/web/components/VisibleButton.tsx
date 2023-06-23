"use client";

import * as React from "react";

import { cn } from "@/lib/utils";
import { Eye, EyeOff } from "lucide-react";

type VisibleButtonProps = React.HTMLAttributes<HTMLButtonElement> & {
  isVisible: boolean;
  setIsVisible: (visible: boolean) => void;
};

export function VisibleButton({
  className,
  isVisible,
  setIsVisible,
  ...props
}: VisibleButtonProps) {
  React.useEffect(() => {
    setTimeout(() => {
      setIsVisible(false);
    }, 10000);
  }, [isVisible]);

  return (
    <button
      type="button"
      className={cn(
        "relative z-20 inline-flex h-8 items-center justify-center rounded-md border-zinc-200 p-2 text-sm font-medium text-zinc-900 transition-all hover:bg-zinc-100 focus:outline-none dark:text-zinc-100 dark:hover:bg-zinc-800",
        className,
      )}
      onClick={() => {
        setIsVisible(!isVisible);
      }}
      {...props}
    >
      <span className="sr-only">Show</span>
      {isVisible ? <Eye className="w-4 h-4" /> : <EyeOff className="w-4 h-4" />}
    </button>
  );
}
