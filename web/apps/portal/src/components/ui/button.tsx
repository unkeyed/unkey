import { Button as ButtonPrimitive } from "@base-ui/react/button";
import { type VariantProps, cva } from "class-variance-authority";
import { cn } from "~/lib/utils";

const buttonVariants = cva(
  "inline-flex cursor-pointer items-center justify-center gap-2 whitespace-nowrap rounded-md font-medium text-sm transition duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-12 focus-visible:ring-offset-1 disabled:pointer-events-none disabled:opacity-50 motion-safe:active:scale-[0.97] [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        default: [
          "relative isolate border border-transparent text-white",
          "[--btn-bg:var(--portal-primary,var(--color-gray-12))] [--btn-border:color-mix(in_oklab,var(--btn-bg)_90%,black)]",
          "bg-(--btn-border)",
          "before:absolute before:inset-0 before:-z-10 before:rounded-[calc(var(--radius-md)-1px)] before:bg-(--btn-bg) before:shadow-sm",
          "after:absolute after:inset-0 after:-z-10 after:rounded-[calc(var(--radius-md)-1px)] after:shadow-[inset_0_1px_rgb(255_255_255/0.15)]",
          "hover:after:bg-white/10 active:after:bg-white/10",
          "disabled:before:shadow-none disabled:after:shadow-none",
        ],
        outline: "border border-primary/15 bg-background shadow-xs hover:bg-gray-2",
        ghost: "text-gray-12 hover:bg-gray-3",
        destructive: "bg-error-9 text-white hover:bg-error-11",
      },
      size: {
        default: "h-8 px-3 has-[>svg:last-child]:pr-2 has-[>svg:first-child]:pl-2",
        sm: "h-6 px-3 text-xs has-[>svg:last-child]:pr-2 has-[>svg:first-child]:pl-2",
        icon: "h-8 w-8 p-0",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
);

type ButtonProps = ButtonPrimitive.Props & VariantProps<typeof buttonVariants>;

export function Button({ className, variant, size, ...props }: ButtonProps) {
  return (
    <ButtonPrimitive className={cn(buttonVariants({ variant, size }), className)} {...props} />
  );
}

export { buttonVariants };
export type { ButtonProps };
