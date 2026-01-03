"use client";
import { Slot } from "@radix-ui/react-slot";
import { type VariantProps, cva } from "class-variance-authority";
import * as React from "react";
import { cn } from "../../lib/utils";
import { AnimatedLoadingSpinner } from "../animated-loading-spinner";

// Hack to populate fumadocs' AutoTypeTable
export type DocumentedButtonProps = VariantProps<typeof buttonVariants> & {
  /**
   * Display a loading spinner instead of the children
   */
  loading?: boolean;

  /**
   * Disables the button
   */
  disabled?: boolean;

  /**
   * Keyboard shortcut to trigger the `onClick` handler
   */
  keyboard?: {
    /**
     * The shortcut displayed on the button
     */
    display: string;
    /**
     * Decide whether the button should be pressed
     * Return true to trigger the callback function.
     * @example: (e)=> e.key === "a"
     */
    trigger: (e: KeyboardEvent) => boolean;
    /**
     * The function to be called
     */
    callback: (e: KeyboardEvent) => void | Promise<void>;
  };

  /**
   * Allows you to use your own component as a button
   */
  asChild?: boolean;

  /**
   * Optional label for screen readers when in loading state
   */
  loadingLabel?: string;
};

const buttonVariants = cva(
  "inline-flex group relative duration-150 items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0 disabled:cursor-not-allowed",
  {
    variants: {
      variant: {
        default: "", // This is only required for mapping from default -> primary. We rely on this for type generation because CVA types are hard to mutate.
        destructive: "", // This is only required for mapping from destructive-> danger. We rely on this for type generation because CVA types are hard to mutate.
        primary: [
          "p-2 text-white dark:text-black bg-accent-12 hover:bg-accent-12/90 focus:hover:bg-accent-12 rounded-md font-medium border border-grayA-4",
          "focus:ring focus:ring-gray-5 focus-visible:outline-none focus:ring-offset-0 drop-shadow-button transform-gpu",
          "disabled:border disabled:border-solid disabled:bg-grayA-6 disabled:border-grayA-4 disabled:text-white/85 disabled:dark:text-white/85",
          "active:bg-accent-12/80",
        ],
        outline: [
          "p-2 text-gray-12 bg-transparent border border-grayA-6 hover:bg-grayA-2 rounded-md",
          "focus:border-grayA-12 focus:ring focus:ring-gray-5 focus-visible:outline-none focus:ring-offset-0 drop-shadow-button transform-gpu",
          "disabled:border disabled:border-solid disabled:border-grayA-5 disabled:text-grayA-7",
          "active:bg-grayA-3",
        ],
        ghost: [
          "p-2 text-gray-12 bg-transparent hover:bg-grayA-4 rounded-md focus:hover:bg-transparent",
          "focus:border-grayA-12 focus:ring focus:ring-gray-5 focus-visible:outline-none focus:ring-offset-0 drop-shadow-button transform-gpu",
          "disabled:border disabled:border-grayA-4 disabled:text-grayA-7",
          "active:bg-grayA-5",
        ],
      },
      // TODO: Remove "square" this in the following iterations. This is only needed for backward compatability
      shape: {
        square: "size-8 p-1",
      },
      color: {
        default: "",
        success: "",
        warning: "",
        danger: "",
        info: "",
      },
      size: {
        // TODO: Remove "icon" this in the following iterations. This is only needed for backward compatability
        icon: "size-6",
        sm: "h-7",
        md: "h-8",
        lg: "h-9",
        xlg: "h-10",
        "2xlg": "h-12",
      },
    },
    defaultVariants: {
      variant: "primary",
      color: "default",
      size: "sm",
    },
    compoundVariants: [
      // Danger
      {
        variant: "primary",
        color: "danger",
        className: [
          "dark:text-white/95 bg-error-9 hover:bg-error-10 rounded-md font-medium focus:hover:bg-error-10 ",
          "focus:border-error-11 focus:ring focus:ring-error-4 focus-visible:outline-none focus:ring-offset-0",
          "disabled:bg-error-6 disabled:text-white/80  disabled:dark:text-white/80",
          "active:bg-error-11",
        ],
      },
      {
        variant: "outline",
        color: "danger",
        className: [
          "text-error-11 bg-transparent border border-grayA-6 hover:bg-grayA-2 font-medium focus:hover:bg-transparent",
          "focus:border-error-11 focus:ring focus:ring-error-4 focus-visible:outline-none focus:ring-offset-0",
          "disabled:text-errorA-7 disabled:border-grayA-5",
          "active:bg-error-3",
        ],
      },
      {
        variant: "ghost",
        color: "danger",
        className: [
          "text-error-11 bg-transparent hover:bg-error-3 rounded-md",
          "focus:border-error-11 focus:ring focus:ring-error-4 focus-visible:outline-none focus:ring-offset-0",
          "disabled:text-error-7",
          "active:bg-error-4",
        ],
      },
      // Warning
      {
        variant: "primary",
        color: "warning",
        className: [
          "dark:text-white/95 bg-warning-8 hover:bg-warning-8/90 rounded-md font-medium focus:hover:bg-warning-8/90",
          "focus:border-warning-11 focus:ring focus:ring-warning-4 focus-visible:outline-none focus:ring-offset-0",
          "disabled:bg-warning-7 disabled:text-white/80  disabled:dark:text-white/80",
          "active:bg-warning-9",
        ],
      },
      {
        variant: "outline",
        color: "warning",
        className: [
          "text-warningA-11 bg-transparent border border-grayA-6 hover:bg-grayA-2 font-medium focus:hover:bg-transparent",
          "focus:border-warning-11 focus:ring focus:ring-warning-4 focus-visible:outline-none focus:ring-offset-0",
          "disabled:text-warningA-7 disabled:border-grayA-5",
          "active:bg-warning-3",
        ],
      },
      {
        variant: "ghost",
        color: "warning",
        className: [
          "text-warning-11 bg-transparent hover:bg-warning-3 rounded-md",
          "focus:border-warning-11 focus:ring focus:ring-warning-4 focus-visible:outline-none focus:ring-offset-0",
          "disabled:text-warning-7",
          "active:bg-warning-4",
        ],
      },
      // Success
      {
        variant: "primary",
        color: "success",
        className: [
          "dark:text-white/95 bg-success-9 hover:bg-success-10 rounded-md font-medium focus:hover:bg-success-10",
          "focus:border-success-11 focus:ring focus:ring-success-4 focus-visible:outline-none focus:ring-offset-0",
          "disabled:bg-success-7 disabled:text-white/80  disabled:dark:text-white/80",
          "active:bg-success-11",
        ],
      },
      {
        variant: "outline",
        color: "success",
        className: [
          "text-success-11 bg-transparent border border-grayA-6 hover:bg-grayA-2 font-medium focus:hover:bg-transparent",
          "focus:border-success-11 focus:ring focus:ring-success-4 focus-visible:outline-none focus:ring-offset-0",
          "disabled:text-successA-7 disabled:border-grayA-5",
          "active:bg-success-3",
        ],
      },
      {
        variant: "ghost",
        color: "success",
        className: [
          "text-success-11 bg-transparent hover:bg-success-3 rounded-md",
          "focus:border-success-11 focus:ring focus:ring-success-4 focus-visible:outline-none focus:ring-offset-0",
          "disabled:text-success-7",
          "active:bg-success-4",
        ],
      },

      // Info
      {
        variant: "primary",
        color: "info",
        className: [
          "dark:text-white/95 bg-info-9 hover:bg-info-10 rounded-md font-medium focus:hover:bg-info-10",
          "focus:border-info-11 focus:ring focus:ring-info-4 focus-visible:outline-none focus:ring-offset-0",
          "disabled:bg-info-7 disabled:text-white/80  disabled:dark:text-white/80",
          "active:bg-info-11",
        ],
      },
      {
        variant: "outline",
        color: "info",
        className: [
          "text-info-11 bg-transparent border border-grayA-6 hover:bg-grayA-2 font-medium focus:hover:bg-transparent",
          "focus:border-info-11 focus:ring focus:ring-info-4 focus-visible:outline-none focus:ring-offset-0",
          "disabled:text-infoA-7 disabled:border-grayA-5",
          "active:bg-info-3",
        ],
      },
      {
        variant: "ghost",
        color: "info",
        className: [
          "text-info-11 bg-transparent hover:bg-info-3 rounded-md",
          "focus:border-info-11 focus:ring focus:ring-info-4 focus-visible:outline-none focus:ring-offset-0",
          "disabled:text-info-7",
          "active:bg-info-4",
        ],
      },
    ],
  },
);

export type ButtonProps = VariantProps<typeof buttonVariants> &
  React.ButtonHTMLAttributes<HTMLButtonElement> & {
    /**
     * Display a loading spinner instead of the children
     */
    loading?: boolean;
    disabled?: boolean;
    /**
     * Keyboard shortcut to trigger the `onClick` handler
     */
    keyboard?: {
      /**
       * The shortcut displayed on the button
       */
      display: string;
      /**
       * Decide whether the button should be pressed
       * Return true to trigger the callback function.
       * @example: (e)=> e.key === "a"
       */
      trigger: (e: KeyboardEvent) => boolean;
      /**
       * The function to be called
       */
      callback: (e: KeyboardEvent) => void | Promise<void>;
    };
    asChild?: boolean;
    /**
     * Optional label for screen readers when in loading state
     */
    loadingLabel?: string;
  };

const keyboardIconVariants = cva(
  "items-center transition duration-150 text-center justify-center shadow-none text-sm flex justify-center font-mono text-xs font-medium border rounded-[5px] h-5 px-1.5 min-w-[24px]",
  {
    variants: {
      variant: {
        default: "bg-gray-4 border-gray-7 text-gray-12",
        primary: "bg-gray-12/10 border-gray-8 text-white dark:text-black group-hover:bg-gray-12/20",
        outline:
          "bg-gray-3 border-gray-6 text-gray-11 group-hover:bg-gray-4 group-hover:border-gray-7",
        ghost:
          "bg-gray-3 border-gray-6 text-gray-11 group-hover:bg-gray-4 group-hover:border-gray-7",
        danger:
          "bg-error-3 border-error-6 text-error-11 group-hover:border-error-7 group-hover:bg-error-4",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);

type ButtonVariant = NonNullable<VariantProps<typeof buttonVariants>["variant"]>;

type ButtonColor = NonNullable<VariantProps<typeof buttonVariants>["color"]>;

const VARIANT_MAP: Record<string, { variant: ButtonVariant; color?: ButtonColor }> = {
  default: { variant: "primary" },
  destructive: { variant: "primary", color: "danger" },
};

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      className,
      variant,
      color = "default",
      size,
      asChild = false,
      loading,
      disabled,
      loadingLabel = "Loading, please wait",
      ...props
    },
    ref,
  ) => {
    let mappedVariant: ButtonVariant = "primary";
    let mappedColor: ButtonColor = color;
    if (variant === null || variant === undefined) {
      mappedVariant = "primary";
    } else if (VARIANT_MAP[variant as keyof typeof VARIANT_MAP]) {
      const mapping = VARIANT_MAP[variant as keyof typeof VARIANT_MAP];
      mappedVariant = mapping.variant;
      if (mapping.color) {
        mappedColor = mapping.color;
      }
    } else {
      mappedVariant = variant as ButtonVariant;
    }
    // Only disable the click behavior, not the visual appearance
    const isClickDisabled = disabled || loading;
    // Keep separate flag for actual visual disabled state
    const isVisuallyDisabled = disabled;
    // Width reference for consistent sizing during loading state
    const buttonRef = React.useRef<HTMLButtonElement>(null);
    const [buttonWidth, setButtonWidth] = React.useState<number | undefined>(undefined);
    // Capture initial width when entering loading state
    React.useEffect(() => {
      if (loading && buttonRef.current && !buttonWidth) {
        setButtonWidth(buttonRef.current.offsetWidth);
      } else if (!loading) {
        setButtonWidth(undefined);
      }
    }, [loading, buttonWidth]);
    // Keyboard handler
    React.useEffect(() => {
      if (!props.keyboard || isClickDisabled) {
        return;
      }
      const down = (e: KeyboardEvent) => {
        if (!props.keyboard?.trigger(e)) {
          return;
        }
        e.preventDefault();
        props.keyboard?.callback(e);
      };
      document.addEventListener("keydown", down);
      return () => document.removeEventListener("keydown", down);
    }, [props.keyboard, isClickDisabled]);
    const Comp = asChild ? Slot : "button";

    return (
      <Comp
        className={cn(
          buttonVariants({
            variant: mappedVariant,
            color: mappedColor,
            size,
            className,
          }),
        )}
        onClick={loading ? undefined : props.onClick}
        disabled={isVisuallyDisabled} // Only apply disabled attribute when explicitly disabled
        aria-disabled={isClickDisabled} // For accessibility, still indicate it can't be clicked
        aria-busy={loading}
        ref={ref}
        {...props}
      >
        {loading && (
          <div
            className="absolute inset-0 flex items-center justify-center w-full h-full transition-opacity duration-200"
            aria-hidden="true"
          >
            <AnimatedLoadingSpinner />
            <span className="sr-only">{loadingLabel}</span>
          </div>
        )}
        <div
          className={cn(
            "w-full h-full flex items-center justify-center gap-2 transition-opacity duration-200",
            {
              "opacity-0": loading,
              "opacity-100": !loading,
            },
          )}
        >
          {props.children}
          {props.keyboard ? (
            <kbd
              className={cn(
                keyboardIconVariants({
                  variant:
                    variant === "primary" ? "primary" : variant === "outline" ? "default" : "ghost",
                }),
              )}
            >
              {props.keyboard.display}
            </kbd>
          ) : null}{" "}
        </div>
      </Comp>
    );
  },
);

// Add CSS for respecting reduced motion preference and adding the spin-slow animation
if (typeof document !== "undefined") {
  const style = document.createElement("style");
  style.textContent = `
    @media (prefers-reduced-motion: reduce) {
      [data-prefers-reduced-motion="respect-motion-preference"] {
        animation: none !important;
        transition: none !important;
      }
    }

    @keyframes spin-slow {
      from {
        transform: rotate(0deg);
      }
      to {
        transform: rotate(360deg);
      }
    }

    .animate-spin-slow {
      animation: spin-slow 1.5s linear infinite;
    }
  `;
  document.head.appendChild(style);
}

Button.displayName = "Button";

export { Button, buttonVariants };
