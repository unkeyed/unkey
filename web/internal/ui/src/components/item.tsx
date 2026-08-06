"use client";

import { useRender } from "@base-ui/react/use-render";
import { type VariantProps, cva } from "class-variance-authority";
import type * as React from "react";
import { cn } from "../lib/utils";

const interactiveClassName =
  "[a&]:cursor-pointer [a&]:hover:bg-grayA-2 [a&]:focus-visible:ring-2 [a&]:focus-visible:ring-grayA-7 [button&]:cursor-pointer [button&]:hover:bg-grayA-2 [button&]:focus-visible:ring-2 [button&]:focus-visible:ring-grayA-7";

const itemVariants = cva(
  `group/item flex w-full items-center gap-3 rounded-lg border border-transparent px-4 py-3 text-left transition-colors focus-visible:outline-hidden ${interactiveClassName}`,
  {
    variants: {
      variant: {
        default: "bg-transparent",
        outline: "border-grayA-4",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);

export type ItemProps = React.HTMLAttributes<HTMLElement> &
  VariantProps<typeof itemVariants> & {
    render?: React.ReactElement;
    ref?: React.Ref<HTMLElement>;
  };

function ItemRenderSlot({
  render,
  ref,
  ...props
}: {
  render: React.ReactElement;
  ref?: React.Ref<HTMLElement>;
} & React.HTMLAttributes<HTMLElement>) {
  return useRender({ render, ref, props });
}

export function Item({ className, variant, render, ref, ...props }: ItemProps) {
  const itemClassName = cn(itemVariants({ variant, className }));

  if (render) {
    return <ItemRenderSlot render={render} ref={ref} className={itemClassName} {...props} />;
  }

  return <div ref={ref as React.Ref<HTMLDivElement>} className={itemClassName} {...props} />;
}

export function ItemMedia({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      aria-hidden="true"
      className={cn(
        "flex size-7 shrink-0 items-center justify-center rounded-md bg-grayA-3 text-gray-11 [&_svg:not([class*='size-'])]:size-3.5",
        className,
      )}
      {...props}
    />
  );
}

export function ItemContent({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("flex min-w-0 flex-1 flex-col gap-px", className)} {...props} />;
}

export function ItemTitle({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cn("text-[13px] font-medium leading-4 text-gray-12", className)} {...props} />
  );
}

export function ItemDescription({
  className,
  ...props
}: React.HTMLAttributes<HTMLParagraphElement>) {
  return <p className={cn("text-xs leading-4 text-gray-11", className)} {...props} />;
}

export function ItemActions({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        "flex shrink-0 items-center gap-2 text-gray-9 transition-colors group-hover/item:text-gray-11 [&_svg:not([class*='size-'])]:size-3.5",
        className,
      )}
      {...props}
    />
  );
}
