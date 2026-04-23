import { type VariantProps, cva } from "class-variance-authority";
import type * as React from "react";
import { cn } from "~/lib/utils";

export function Empty({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center gap-6 px-8 py-16 text-center",
        className,
      )}
      {...props}
    />
  );
}

export function EmptyHeader({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("flex flex-col items-center gap-3", className)} {...props} />;
}

const emptyMediaVariants = cva("flex items-center justify-center rounded-md text-gray-11", {
  variants: {
    variant: {
      default: "",
      icon: "h-12 w-12 border border-gray-6 bg-grayA-2 [&_svg]:size-5",
    },
  },
  defaultVariants: { variant: "default" },
});

export function EmptyMedia({
  className,
  variant,
  ...props
}: React.HTMLAttributes<HTMLDivElement> & VariantProps<typeof emptyMediaVariants>) {
  return <div className={cn(emptyMediaVariants({ variant }), className)} {...props} />;
}

export function EmptyTitle({ className, ...props }: React.HTMLAttributes<HTMLHeadingElement>) {
  return <h3 className={cn("text-base font-medium text-gray-12", className)} {...props} />;
}

export function EmptyDescription({
  className,
  ...props
}: React.HTMLAttributes<HTMLParagraphElement>) {
  return <p className={cn("max-w-sm text-sm text-gray-11", className)} {...props} />;
}

export function EmptyContent({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("flex flex-col items-center gap-2", className)} {...props} />;
}
