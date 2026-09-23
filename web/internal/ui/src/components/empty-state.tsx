import { type VariantProps, cva } from "class-variance-authority";
import type * as React from "react";
import { cn } from "../lib/utils";

const emptyStateVariants = cva(
  "flex w-full flex-col items-center justify-center gap-3 px-4 py-10 text-center",
  {
    variants: {
      frame: {
        dashed: "rounded-lg border border-dashed bg-background",
        none: "",
      },
    },
    defaultVariants: {
      frame: "dashed",
    },
  },
);

type EmptyStateProps = React.ComponentProps<"div"> & VariantProps<typeof emptyStateVariants>;

function EmptyState({ className, frame, ...props }: EmptyStateProps) {
  return (
    <div
      data-slot="empty-state"
      className={cn(emptyStateVariants({ frame, className }))}
      {...props}
    />
  );
}

function EmptyStateIcon({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div aria-hidden="true" className={cn("text-gray-9 [&_svg]:size-6", className)} {...props} />
  );
}

function EmptyStateHeader({ className, ...props }: React.ComponentProps<"div">) {
  return <div className={cn("flex flex-col items-center", className)} {...props} />;
}

function EmptyStateTitle({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div className={cn("text-[15px] font-semibold leading-6 text-gray-12", className)} {...props} />
  );
}

function EmptyStateDescription({ className, ...props }: React.ComponentProps<"p">) {
  return (
    <p
      className={cn("max-w-md text-[13px] leading-5 text-balance text-gray-11", className)}
      {...props}
    />
  );
}

function EmptyStateActions({ className, ...props }: React.ComponentProps<"div">) {
  return <div className={cn("flex shrink-0 items-center gap-2", className)} {...props} />;
}

export {
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
};
