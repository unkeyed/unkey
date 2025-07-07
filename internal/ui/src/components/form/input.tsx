import { type VariantProps, cva } from "class-variance-authority";
import * as React from "react";
import { cn } from "../../lib/utils";

const inputVariants = cva(
  "flex min-h-9 w-full rounded-lg text-[13px] leading-5 transition-colors duration-300 disabled:cursor-not-allowed disabled:opacity-50 placeholder:text-grayA-8 text-grayA-12",
  {
    variants: {
      variant: {
        default: [
          "border border-gray-5 hover:border-gray-8 bg-gray-2 dark:bg-black",
          "focus:border focus:border-accent-12 focus:ring focus:ring-gray-5 focus-visible:outline-none focus:ring-offset-0",
        ],
        success: [
          "border border-success-9 hover:border-success-10 bg-gray-2 dark:bg-black",
          "focus:border-success-8 focus:ring focus:ring-success-4 focus-visible:outline-none ",
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

const inputWrapperVariants = cva("relative flex items-center w-full", {
  variants: {
    variant: {
      default: "text-grayA-12",
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

const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, variant, type, leftIcon, rightIcon, prefix, wrapperClassName, ...props }, ref) => {
    const prefixRef = React.useRef<HTMLSpanElement>(null);
    const [prefixWidth, setPrefixWidth] = React.useState(0);

    React.useEffect(() => {
      if (prefix && prefixRef.current) {
        setPrefixWidth(prefixRef.current.offsetWidth);
      }
    }, [prefix]);

    return (
      <div className={cn(inputWrapperVariants({ variant }), wrapperClassName)}>
        {leftIcon && (
          <div className="absolute left-3 flex items-center pointer-events-none">{leftIcon}</div>
        )}
        {prefix && (
          <span
            ref={prefixRef}
            className="absolute left-3 flex items-center pointer-events-none text-[13px] leading-5 opacity-40 select-none"
            style={{ zIndex: 1 }}
          >
            {prefix}
          </span>
        )}
        <input
          type={type}
          className={cn(
            inputVariants({ variant, className }),
            "px-3 py-2",
            leftIcon && "pl-9",
            rightIcon && "pr-9",
            prefix && !leftIcon && "pl-3",
          )}
          style={{
            paddingLeft: prefix && !leftIcon ? `${prefixWidth + 12}px` : undefined,
          }}
          ref={ref}
          {...props}
        />
        {rightIcon && <div className="absolute right-3 flex items-center">{rightIcon}</div>}
      </div>
    );
  },
);

Input.displayName = "Input";

export { Input, type InputProps, type DocumentedInputProps };
