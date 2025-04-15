import * as React from "react";

import { cn } from "@/lib/utils";
import type { LucideIcon } from "lucide-react";

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  startIcon?: LucideIcon;
  endIcon?: LucideIcon;
}

const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, type, startIcon, endIcon, ...props }, ref) => {
    const StartIcon = startIcon;
    const EndIcon = endIcon;

    return (
      <div className="w-full">
        {StartIcon && (
          <div className="absolute left-1.5 top-1/2 transform -translate-y-1/2" aria-hidden="true">
            <StartIcon size={18} className="text-muted-foreground" />
          </div>
        )}
        <input
          type={type}
          className={cn(
            "flex h-8 w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus:border-primary placeholder:text-content-subtle focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50",
            startIcon ? "pl-8" : "",
            endIcon ? "pr-8" : "",
            className,
          )}
          ref={ref}
          {...props}
        />
        {EndIcon && (
          <div className="absolute right-3 top-1/2 transform -translate-y-1/2" aria-hidden="true">
            <EndIcon className="text-muted-foreground" size={18} />
          </div>
        )}
      </div>
    );
  },
);
Input.displayName = "Input";

export { Input };
