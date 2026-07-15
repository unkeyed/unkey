import { type VariantProps, cva } from "class-variance-authority";
import type * as React from "react";
import { cn } from "~/lib/utils";

const alertVariants = cva(
  [
    "grid w-full grid-cols-[auto_1fr] items-start gap-x-3 gap-y-0.5 rounded-lg border p-3 text-xs",
    "[&>svg]:row-span-2 [&>svg]:mt-0.5 [&>svg]:size-4 [&>svg]:shrink-0",
  ],
  {
    variants: {
      variant: {
        default: "border-primary/15 bg-background text-gray-12 [&>svg]:text-gray-11",
        destructive: [
          "border-red-200 bg-red-50 text-red-900",
          "[&>svg]:text-red-600",
          "[&>[data-slot=description]]:text-red-800",
        ],
        warning: [
          "border-amber-200 bg-amber-50 text-amber-900",
          "[&>svg]:text-amber-600",
          "[&>[data-slot=description]]:text-amber-800",
        ],
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);

type AlertProps = React.HTMLAttributes<HTMLDivElement> & VariantProps<typeof alertVariants>;

export function Alert({ className, variant, ...props }: AlertProps) {
  return <div role="alert" className={cn(alertVariants({ variant }), className)} {...props} />;
}

export function AlertTitle({ className, ...props }: React.HTMLAttributes<HTMLParagraphElement>) {
  return <p data-slot="title" className={cn("font-semibold", className)} {...props} />;
}

export function AlertDescription({
  className,
  ...props
}: React.HTMLAttributes<HTMLParagraphElement>) {
  return <p data-slot="description" className={cn(className)} {...props} />;
}
