"use client";

import { useRender } from "@base-ui/react/use-render";
import { type VariantProps, cva } from "class-variance-authority";
import type * as React from "react";
import { cn } from "../lib/utils";
import { Separator } from "./separator";

const interactiveClassName =
  "[a&]:cursor-pointer [a&]:hover:bg-grayA-2 [a&]:focus-visible:ring-2 [a&]:focus-visible:ring-grayA-7 [button&]:cursor-pointer [button&]:hover:bg-grayA-2 [button&]:focus-visible:ring-2 [button&]:focus-visible:ring-grayA-7";

const rowClassName = "flex w-full items-center gap-3 px-4 py-3 text-left";

const itemVariants = cva(
  `group/item ${rowClassName} rounded-lg border border-transparent transition-colors focus-visible:outline-hidden ${interactiveClassName}`,
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

const itemGroupVariants = cva("flex w-full flex-col", {
  variants: {
    variant: {
      default: "",
      outline: "rounded-lg border border-grayA-4",
    },
  },
  defaultVariants: {
    variant: "default",
  },
});

export type ItemGroupProps = React.HTMLAttributes<HTMLDivElement> &
  VariantProps<typeof itemGroupVariants>;

export function ItemGroup({ className, variant, ...props }: ItemGroupProps) {
  return (
    <div
      data-slot="item-group"
      className={cn(itemGroupVariants({ variant, className }))}
      {...props}
    />
  );
}

export function ItemHeader({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div data-slot="item-header" className={cn(rowClassName, className)} {...props} />;
}

export function ItemFooter({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      data-slot="item-footer"
      className={cn(rowClassName, "text-xs text-gray-11", className)}
      {...props}
    />
  );
}

export function ItemSeparator({ className, ...props }: React.ComponentProps<typeof Separator>) {
  return (
    <Separator data-slot="item-separator" className={cn("bg-grayA-4", className)} {...props} />
  );
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
    <div className={cn("text-[13px] font-[450] leading-4 text-gray-12", className)} {...props} />
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
      data-slot="item-actions"
      className={cn(
        "flex shrink-0 items-center gap-2 text-[13px] leading-4 text-gray-12 [&_svg]:text-gray-9 [&_svg]:transition-colors group-hover/item:[&_svg]:text-gray-11 [&_svg:not([class*='size-'])]:size-3.5",
        className,
      )}
      {...props}
    />
  );
}
