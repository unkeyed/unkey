"use client";

import { type VariantProps, cva } from "class-variance-authority";
// biome-ignore lint/style/useImportType: Biome wants this
import React from "react";
import { forwardRef, useEffect, useRef, useState } from "react";
import { cn } from "../../lib/utils";

// Layout constants
const BASE_PADDING = 8; // 8px base padding
const ICON_SPACE = 36; // 36px total space for icons
const PREFIX_BUFFER = 1; // 1px gap after prefix

const inputVariants = cva(
  "flex min-h-9 w-full rounded-lg text-[13px] leading-5 transition-colors duration-300 disabled:cursor-not-allowed disabled:opacity-50 placeholder:text-grayA-8 text-grayA-12",
  {
    variants: {
      variant: {
        default: [
          "border border-gray-5 hover:border-gray-8 bg-gray-2 dark:bg-black",
          "focus:border focus:border-accent-12 focus:ring focus:ring-gray-5 focus-visible:outline-none focus:ring-offset-0",
        ],
        ghost: [
          "border border-transparent bg-transparent focus:border focus:border-accent-12 focus:ring focus:ring-gray-5 focus-visible:outline-none focus:ring-offset-0",
        ],
        success: [
          "border border-success-9 hover:border-success-10 bg-gray-2 dark:bg-black",
          "focus:border-success-8 focus:ring focus:ring-success-4 focus-visible:outline-none",
        ],
        warning: [
          "border border-warning-9 hover:border-warning-10 bg-gray-2 dark:bg-black",
          "focus:border-warning-8 focus:ring focus:ring-warning-4 focus-visible:outline-none",
        ],
        error: [
          "border border-error-9 hover:border-error-10 bg-gray-2 dark:bg-black",
          "focus:border-error-8 focus:ring focus:ring-error-4 focus-visible:outline-none",
        ],
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);

const wrapperVariants = cva("relative flex items-center w-full", {
  variants: {
    variant: {
      default: "text-grayA-12",
      ghost: "text-grayA-12",
      success: "text-success-11",
      warning: "text-warning-11",
      error: "text-error-11",
    },
  },
  defaultVariants: {
    variant: "default",
  },
});

// Hack to populate fumadocs' AutoTypeTable
type DocumentedInputProps = VariantProps<typeof inputVariants> & {
  leftIcon?: React.ReactNode;
  rightIcon?: React.ReactNode;
  prefix?: string;
  wrapperClassName?: string;
};

type InputProps = DocumentedInputProps & React.InputHTMLAttributes<HTMLInputElement>;

const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ className, variant, leftIcon, rightIcon, prefix, wrapperClassName, ...props }, ref) => {
    const prefixRef = useRef<HTMLElement>(null) as React.RefObject<HTMLElement>;
    const [prefixWidth, setPrefixWidth] = useState(0);

    // biome-ignore lint/correctness/useExhaustiveDependencies: prefixRef is stable and shouldn't be in deps
    useEffect(() => {
      if (prefix && prefixRef.current) {
        setPrefixWidth(prefixRef.current.offsetWidth);
      } else {
        setPrefixWidth(0);
      }
    }, [prefix]);

    // Calculate input padding
    const getLeftPadding = (): string => {
      if (leftIcon) {
        return `${ICON_SPACE}px`;
      }
      if (prefix) {
        return `${BASE_PADDING + prefixWidth + PREFIX_BUFFER}px`;
      }
      return `${BASE_PADDING}px`;
    };

    const getRightPadding = (): string => {
      return rightIcon ? `${ICON_SPACE}px` : `${BASE_PADDING}px`;
    };

    return (
      <div className={cn(wrapperVariants({ variant }), wrapperClassName)}>
        {/* Left Icon */}
        {leftIcon && (
          <div className="absolute left-3 flex items-center pointer-events-none z-10">
            {leftIcon}
          </div>
        )}

        {/* Prefix */}
        {prefix && (
          <span
            ref={prefixRef}
            className="absolute left-2 flex items-center pointer-events-none text-[13px] leading-5 opacity-40 select-none z-10"
          >
            {prefix}
          </span>
        )}

        {/* Input */}
        <input
          ref={ref}
          className={cn(inputVariants({ variant }), "py-2", className)}
          style={{
            paddingLeft: getLeftPadding(),
            paddingRight: getRightPadding(),
          }}
          {...props}
        />

        {/* Right Icon */}
        {rightIcon && <div className="absolute right-3 flex items-center z-10">{rightIcon}</div>}
      </div>
    );
  },
);

Input.displayName = "Input";

export { Input, type InputProps, type DocumentedInputProps };
