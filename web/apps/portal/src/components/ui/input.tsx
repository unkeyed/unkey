import type * as React from "react";
import { cn } from "~/lib/utils";

type InputProps = React.InputHTMLAttributes<HTMLInputElement>;

export function Input({ className, type = "text", ...props }: InputProps) {
  return (
    <input
      type={type}
      className={cn(
        "flex h-8 w-full rounded-md border border-primary/15 shadow-xs bg-background px-3 text-gray-12 text-sm transition-colors placeholder:text-gray-10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-12 focus-visible:ring-offset-1 disabled:cursor-not-allowed disabled:opacity-50",
        className,
      )}
      {...props}
    />
  );
}

export type { InputProps };
