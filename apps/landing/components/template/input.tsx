import * as React from "react";

import { cn } from "@/lib/utils";
import { Search } from "lucide-react";

export type InputProps = React.InputHTMLAttributes<HTMLInputElement>;

const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, type, ...props }, ref) => {
    return (
      <input
        type={type}
        className={cn(
          "flex h-10 w-full rounded-md border border-border bg-transparent px-3 py-2 text-sm focus:border-white file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-white/40 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50",
          className,
        )}
        ref={ref}
        {...props}
      />
    );
  },
);
Input.displayName = "Input";

export { Input };

const SearchInput = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, type, ...props }, ref) => {
    return (
      <div
        className={cn(
          "flex flex-row h-10 w-full rounded-md border border-border p-0 m-0 bg-transparent focus:border-white file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-white/40 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50",
          className,
        )}
      >
        <Search className="h-full py-2 mx-2 text-white" />
        <input
          type={type}
          ref={ref}
          {...props}
          className="bg-transparent text-white/60 p-0 m-0 w-full border-none h-full"
        />
      </div>
    );
  },
);
Input.displayName = "Input";

export { SearchInput };
