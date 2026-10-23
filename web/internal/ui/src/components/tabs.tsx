"use client";

import { Tabs as TabsPrimitive } from "@base-ui/react/tabs";
import * as React from "react";
import { cva } from "../lib/cva";
import { cn } from "../lib/utils";

const Tabs = TabsPrimitive.Root;

type TabsVariant = "default" | "underline";

const TabsVariantContext = React.createContext<TabsVariant>("default");

const tabsListVariants = cva("inline-flex items-center", {
  variants: {
    variant: {
      default: "h-9 justify-center rounded-lg bg-grayA-3 p-1 text-grayA-11",
      underline: "w-full justify-start border-b text-gray-11",
    },
  },
  defaultVariants: { variant: "default" },
});

const tabsTriggerVariants = cva(
  "inline-flex items-center justify-center whitespace-nowrap text-sm font-medium focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-5 disabled:cursor-not-allowed disabled:opacity-50 aria-disabled:cursor-not-allowed aria-disabled:opacity-50",
  {
    variants: {
      variant: {
        default:
          "rounded-md px-3 py-1 ring-offset-0 transition-all duration-150 hover:bg-grayA-2 data-active:bg-background data-active:text-grayA-12 data-active:shadow-sm",
        underline:
          "cursor-pointer border-b-2 border-transparent px-4 py-2 transition-colors data-active:border-gray-12 data-active:text-gray-12",
      },
    },
    defaultVariants: { variant: "default" },
  },
);

function TabsList({
  className,
  variant = "default",
  ref,
  ...props
}: React.ComponentPropsWithoutRef<typeof TabsPrimitive.List> & {
  variant?: TabsVariant;
  ref?: React.Ref<React.ComponentRef<typeof TabsPrimitive.List>>;
}) {
  return (
    <TabsVariantContext.Provider value={variant}>
      <TabsPrimitive.List
        ref={ref}
        // Radix parity: Radix activated tabs as arrow-key focus moved; Base UI
        // defaults to manual activation (Enter/Space). Restore automatic activation.
        activateOnFocus
        className={cn(tabsListVariants({ variant }), className)}
        {...props}
      />
    </TabsVariantContext.Provider>
  );
}

function TabsTrigger({
  className,
  ref,
  ...props
}: React.ComponentPropsWithoutRef<typeof TabsPrimitive.Tab> & {
  ref?: React.Ref<React.ComponentRef<typeof TabsPrimitive.Tab>>;
}) {
  const variant = React.useContext(TabsVariantContext);
  return (
    <TabsPrimitive.Tab
      ref={ref}
      className={cn(tabsTriggerVariants({ variant }), className)}
      {...props}
    />
  );
}

function TabsContent({
  className,
  ref,
  ...props
}: React.ComponentPropsWithoutRef<typeof TabsPrimitive.Panel> & {
  ref?: React.Ref<React.ComponentRef<typeof TabsPrimitive.Panel>>;
}) {
  return (
    <TabsPrimitive.Panel
      ref={ref}
      className={cn(
        "mt-2 ring-offset-0 focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-5",
        className,
      )}
      {...props}
    />
  );
}

export { Tabs, TabsList, TabsTrigger, TabsContent };
