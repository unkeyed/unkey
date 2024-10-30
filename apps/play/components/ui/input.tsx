import * as React from "react";

import { cn } from "@/lib/utils";

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {}

const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, type, ...props }, ref) => {
    return (
      <input
        type={type}
        className={cn(
          "flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50",
          className,
        )}
        ref={ref}
        {...props}
      />
    );
  },
);
Input.displayName = "Input";

export interface NamedInputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label: string;
}

const NamedInput = React.forwardRef<HTMLInputElement, NamedInputProps>(
  ({ className, type, label, ...props }, ref) => {
    return (
      <div
        className={cn(
          "flex h-9 w-full rounded-md border border-input bg-transparent text-sm shadow-sm has-[:focus-visible]:ring-1 has-[:focus-visible]:ring-ring has-[:disabled]:cursor-not-allowed has-[:disabled]:opacity-50",
          className,
        )}
      >
        <label
          htmlFor={props.id}
          className="flex shrink-0 flex-grow-0 items-center w-fit h-full px-3 bg-black rounded-l-md"
        >
          {label}
        </label>

        <input
          type={type}
          className={cn(
            "flex w-full bg-transparent px-3 placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-0 border-l",
          )}
          ref={ref}
          {...props}
        />
      </div>
    );
  },
);
Input.displayName = "NamedInput";

export { Input, NamedInput };
