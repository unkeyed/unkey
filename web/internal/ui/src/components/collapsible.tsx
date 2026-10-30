"use client";

import { Collapsible as CollapsiblePrimitive } from "@base-ui/react/collapsible";
import { IconChevronDownOutline12 } from "@unkey/icons";
import * as React from "react";
import { cn } from "../lib/utils";
import { Item, ItemActions, ItemContent, ItemDescription, ItemMedia, ItemTitle } from "./item";

const Collapsible = CollapsiblePrimitive.Root;

function CollapsibleChevron() {
  return (
    <IconChevronDownOutline12
      aria-hidden="true"
      className="size-3 shrink-0 text-gray-10 transition-transform duration-200 ease-out in-data-[panel-open]:rotate-180 motion-reduce:transition-none"
    />
  );
}

function CollapsibleTrigger({
  className,
  children,
  ...props
}: Omit<React.ComponentProps<typeof CollapsiblePrimitive.Trigger>, "className"> & {
  className?: string;
}) {
  return (
    <CollapsiblePrimitive.Trigger className={cn("group", className)} {...props}>
      {children}
      <CollapsibleChevron />
    </CollapsiblePrimitive.Trigger>
  );
}

function CollapsibleContent({
  className,
  ...props
}: Omit<React.ComponentProps<typeof CollapsiblePrimitive.Panel>, "className"> & {
  className?: string;
}) {
  return (
    <CollapsiblePrimitive.Panel
      keepMounted
      className={cn(
        "h-(--collapsible-panel-height) overflow-hidden transition-[height] duration-200 ease-out data-[ending-style]:h-0 data-[starting-style]:h-0 motion-reduce:duration-0",
        className,
      )}
      {...props}
    />
  );
}

type CollapsibleItemProps = {
  icon: React.ReactNode;
  title: React.ReactNode;
  description?: React.ReactNode;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  children: React.ReactNode;
};

function CollapsibleItem({
  icon,
  title,
  description,
  open,
  onOpenChange,
  children,
}: CollapsibleItemProps) {
  const rootRef = React.useRef<HTMLDivElement>(null);
  const revealPending = React.useRef(false);
  const reveal = (behavior: ScrollBehavior) =>
    rootRef.current?.scrollIntoView({ block: "nearest", behavior });

  return (
    <CollapsiblePrimitive.Root
      ref={rootRef}
      open={open}
      onOpenChange={(next) => {
        onOpenChange(next);
        revealPending.current = false;
        if (!next) {
          return;
        }
        if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
          requestAnimationFrame(() => reveal("auto"));
        } else {
          revealPending.current = true;
        }
      }}
      className="scroll-mb-6 rounded-lg border border-grayA-4"
    >
      <Item render={<CollapsiblePrimitive.Trigger />}>
        <ItemMedia>{icon}</ItemMedia>
        <ItemContent>
          <ItemTitle>{title}</ItemTitle>
          {description ? <ItemDescription>{description}</ItemDescription> : null}
        </ItemContent>
        <ItemActions className="[&_svg]:transition-[color,transform]">
          <CollapsibleChevron />
        </ItemActions>
      </Item>
      <CollapsibleContent
        onTransitionEnd={(e) => {
          if (
            e.target === e.currentTarget &&
            e.propertyName === "height" &&
            revealPending.current
          ) {
            revealPending.current = false;
            reveal("smooth");
          }
        }}
      >
        <div className="border-t border-grayA-4">{children}</div>
      </CollapsibleContent>
    </CollapsiblePrimitive.Root>
  );
}

export { Collapsible, CollapsibleContent, CollapsibleItem, CollapsibleTrigger };
