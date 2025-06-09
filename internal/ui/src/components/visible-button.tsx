"use client";

import { Eye, EyeOff } from "lucide-react";
// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import React, { useEffect } from "react";
import { cn } from "../lib/utils";

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
  useEffect(() => {
    if (!isVisible) {
      return;
    }
    const timer = setTimeout(() => {
      setIsVisible(false);
    }, 10000);
    return () => clearTimeout(timer);
  }, [setIsVisible, isVisible]);

  return (
    <button
      type="button"
      className={cn(
        "relative z-20 inline-flex h-6 w-6 items-center justify-center rounded-md border-gray-200 p-1 text-sm font-medium text-gray-900 transition-all hover:bg-gray-100 focus:outline-none dark:text-gray-100 dark:hover:bg-gray-800",
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
