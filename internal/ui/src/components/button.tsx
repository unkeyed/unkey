"use client";
import { Slot } from "@radix-ui/react-slot";
import { type VariantProps, cva } from "class-variance-authority";
import { Loader } from "lucide-react";
import * as React from "react";
import { cn } from "../lib/utils";

const buttonVariants = cva(
  "inline-flex group relative transition duration-150 whitespace-nowrap tracking-normal rounded-lg  font-medium transition-colors disabled:pointer-events-none focus:outline-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        default:
          "bg-gray-3 hover:bg-gray-4 focus-visible:bg-gray-5 text-accent-12 border border-gray-6 hover:border-gray-8 ring-2 ring-transparent focus-visible:ring-gray-7 focus-visible:border-gray-7",
        primary:
          "bg-gray-12 hover:bg-gray-1  text-accent-1 hover:text-accent-12 border border-black dark:border-white hover:border-gray-4 ring-2 ring-transparent focus-visible:ring-gray-7 focus-visible:border-gray-3 drop-shadow-button duration-250",
        destructive:
          "text-error-9 border border-gray-6 hover:border-error-8 hover:bg-error-4 focus-visible:border-error-9 ring-2 ring-transparent focus-visible:ring-error-3",
        ghost:
          "text-accent-12 hover:bg-gray-3 ring-2 ring-transparent focus-visible:ring-accent-12",
      },
      size: {
        default: "h-8 px-3 py-1 text-sm",
        icon: "size-6 p-1",
      },
      shape: {
        square: "size-8 p-1",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
);

const keyboardIconVariants = cva(
  "items-center transition duration-150 text-center items-center justify-center shadow-none size-5 flex justify-center  font-mono text-xs font-medium border   rounded-[5px]",
  {
    variants: {
      variant: {
        default: "bg-gray-3 border-gray-6 text-accent-12",
        primary:
          "duration-250 bg-black/10 border-gray-11 text-accent-1 group-hover:text-accent-12 group-hover:bg-gray-3 group-hover:border-gray-6",
        destructive: "bg-gray-1 border-gray-6 text-accent-12 group-hover:border-error-8",
        ghost: "border-gray-6 text-accent-12 ",
      },
    },
    defaultVariants: {
      variant: "default",
    },
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
  };

const Button: React.FC<ButtonProps> = ({
  className,
  variant,
  size,
  shape,
  asChild = false,
  loading,
  ...props
}) => {
  React.useEffect(() => {
    if (!props.keyboard) {
      return;
    }
    const down = (e: KeyboardEvent) => {
      if (!props.keyboard!.trigger(e)) {
        return;
      }
      e.preventDefault();

      props.keyboard!.callback(e);
    };
    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, [props.keyboard]);

  const Comp = asChild ? Slot : "button";

  return (
    <Comp
      className={cn(buttonVariants({ variant, size, shape, className }))}
      onClick={props.onClick}
      {...props}
    >
      {loading ? (
        <div
          className={cn("inset-0 absolute flex justify-center items-center w-full h-full", {
            "opacity-0": !loading,
            "opacity-100": loading,
          })}
        >
          <Loader className="animate-spin " />
        </div>
      ) : null}
      <div
        className={cn(
          "duration-200 w-full h-full transition-opacity flex items-center justify-center gap-2",
          {
            "opacity-100": !loading,
            "opacity-0": loading,
          },
        )}
      >
        {props.children}
        {props.keyboard ? (
          <kbd className={cn(keyboardIconVariants({ variant }))}>{props.keyboard.display}</kbd>
        ) : null}
      </div>
    </Comp>
  );
};
Button.displayName = "Button";

export { Button, buttonVariants };
