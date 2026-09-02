import { Tabs as TabsPrimitive } from "@base-ui/react/tabs";
import type * as React from "react";
import { cn } from "../lib/utils";

const Tabs = TabsPrimitive.Root;

function TabsList({
  className,
  ref,
  ...props
}: React.ComponentPropsWithoutRef<typeof TabsPrimitive.List> & {
  ref?: React.Ref<React.ComponentRef<typeof TabsPrimitive.List>>;
}) {
  return (
    <TabsPrimitive.List
      ref={ref}
      // Radix parity: Radix activated tabs as arrow-key focus moved; Base UI
      // defaults to manual activation (Enter/Space). Restore automatic activation.
      activateOnFocus
      className={cn(
        "inline-flex h-9 items-center justify-center rounded-lg bg-gray-2 p-1 text-grayA-11",
        className,
      )}
      {...props}
    />
  );
}

function TabsTrigger({
  className,
  ref,
  ...props
}: React.ComponentPropsWithoutRef<typeof TabsPrimitive.Tab> & {
  ref?: React.Ref<React.ComponentRef<typeof TabsPrimitive.Tab>>;
}) {
  return (
    <TabsPrimitive.Tab
      ref={ref}
      className={cn(
        "inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-sm font-medium ring-offset-0 transition-all duration-150 focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-5 disabled:cursor-not-allowed disabled:opacity-50 aria-disabled:cursor-not-allowed aria-disabled:opacity-50 hover:bg-grayA-2 data-active:bg-white dark:data-active:bg-black data-active:text-grayA-12 data-active:shadow-sm",
        className,
      )}
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
