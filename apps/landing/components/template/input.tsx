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
          "focus-within:border-white flex items-center gap-2 h-10 w-full rounded-md border border-border p-0 m-0 bg-transparent focus:border-white file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-white/40 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50 text-sm",
          className,
        )}
      >
        <Search className="w-6 h-6 mx-3 text-white " />
        <input
          type={type}
          ref={ref}
          {...props}
          className="w-full h-full bg-transparent border-none text-white/60 focus:outline-none"
        />
      </div>
    );
  },
);
Input.displayName = "Input";

export { SearchInput };
