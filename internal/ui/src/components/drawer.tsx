"use client";

/* -----------------------------------------------------------------------------
 * Imports
 * ----------------------------------------------------------------------------*/

import * as React from "react";
import { Drawer as Vaul } from "vaul";
import { cn } from "../lib/utils";

/* -----------------------------------------------------------------------------
 *  Extend Drawer
 * ----------------------------------------------------------------------------*/

const DrawerRoot = Vaul.Root;
const DrawerTrigger = Vaul.Trigger;
const DrawerPortal = Vaul.Portal;
const DrawerOverlay = Vaul.Overlay;
const DrawerTitle = Vaul.Title;
const DrawerDescription = Vaul.Description;
const DrawerNested = Vaul.NestedRoot;

/* ----------------------------------------------------------------------------
 * Drawer - Content
 * ---------------------------------------------------------------------------*/

const CONTENT_NAME = "DrawerContent";

type DrawerContentElement = React.ElementRef<typeof Vaul.Content>;
type DrawerContentProps = React.ComponentPropsWithoutRef<typeof Vaul.Content>;

const DrawerContent = React.forwardRef<DrawerContentElement, DrawerContentProps>((props, ref) => {
  const { className, children, ...contentProps } = props;

  return (
    <DrawerPortal>
      <DrawerOverlay className="fixed inset-0 bg-background/60" />
      <Vaul.Content
        {...contentProps}
        ref={ref}
        className={cn(
          "bg-gray-1 border border-gray-6 flex flex-col fixed bottom-0 left-0 right-0 max-h-[82vh] rounded-t-xl drop-shadow-2xl transform-gpu",
          className,
        )}
      >
        {children}
      </Vaul.Content>
    </DrawerPortal>
  );
});

DrawerContent.displayName = CONTENT_NAME;

/* ----------------------------------------------------------------------------
 * Exports
 * ---------------------------------------------------------------------------*/

export const Drawer = Object.assign({
  Root: DrawerRoot,
  Trigger: DrawerTrigger,
  Content: DrawerContent,
  Title: DrawerTitle,
  Description: DrawerDescription,
  Nested: DrawerNested,
});
