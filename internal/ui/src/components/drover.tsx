"use client";

import { Slot } from "@radix-ui/react-slot";
import { useControllableState } from "@radix-ui/react-use-controllable-state";
import React from "react";
import { Drawer } from "../components/drawer";
import { useIsMobile } from "../hooks/use-mobile";
import { createContext } from "../lib/create-context";
import { cn } from "../lib/utils";
import { Popover, PopoverContent, PopoverTrigger } from "./dialog/popover";

type PrimitiveDivProps = React.ComponentPropsWithoutRef<"div">;
type PrimitiveButtonElement = React.ElementRef<"button">;
type PrimitiveButtonProps = React.ComponentPropsWithoutRef<"button">;

/* ----------------------------------------------------------------------------
 * Component Drover:Root
 * --------------------------------------------------------------------------*/

type DroverContextValue = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  isMobile: boolean | undefined;
};
export interface DroverProps extends PrimitiveDivProps {
  children?: React.ReactNode;
  open?: boolean;
  defaultOpen?: boolean;
  onOpenChange?(open: boolean): void;
}

const ROOT_NAME = "Root";

const [DroverProvider, useDroverContext] = createContext<DroverContextValue>(ROOT_NAME);

const Root: React.FC<DroverProps> = (props) => {
  const { open: openProp, defaultOpen, onOpenChange, children } = props;
  // Default to false (desktop) to prevent hydration mismatches and layout shifts
  const isMobile = useIsMobile({ defaultValue: false });
  const [open, setOpen] = useControllableState({
    prop: openProp,
    defaultProp: defaultOpen ?? false,
    onChange: onOpenChange,
    caller: ROOT_NAME,
  });
  const RootComponent = isMobile ? Drawer.Root : Popover;

  return (
    <DroverProvider open={open} onOpenChange={setOpen} isMobile={isMobile}>
      <RootComponent open={open} onOpenChange={setOpen}>
        {children}
      </RootComponent>
    </DroverProvider>
  );
};

Root.displayName = ROOT_NAME;

/* ----------------------------------------------------------------------------
 * Component Drover:Trigger
 * --------------------------------------------------------------------------*/

type TriggerProps = PrimitiveButtonProps & {
  asChild?: boolean;
};
const TRIGGER_NAME = "Trigger";

const Trigger = React.forwardRef<PrimitiveButtonElement, TriggerProps>((props, ref) => {
  const { children, asChild = false, ...triggerProps } = props;
  const { isMobile } = useDroverContext(TRIGGER_NAME);
  const TriggerComponent = isMobile ? Drawer.Trigger : PopoverTrigger;

  return (
    <TriggerComponent ref={ref} asChild={asChild} {...triggerProps}>
      {children}
    </TriggerComponent>
  );
});

Trigger.displayName = TRIGGER_NAME;

/* ----------------------------------------------------------------------------
 * Component Drover:Content
 * --------------------------------------------------------------------------*/

type PopoverContentProps = React.ComponentPropsWithoutRef<typeof PopoverContent>;
type DrawerContentProps = React.ComponentPropsWithoutRef<typeof Drawer.Content>;

type ContentProps = PopoverContentProps | DrawerContentProps;
type ContentElement =
  | React.ElementRef<typeof PopoverContent>
  | React.ElementRef<typeof Drawer.Content>;
const CONTENT_NAME = "Content";

const Content = React.forwardRef<ContentElement, ContentProps>((props, ref) => {
  const { children, onKeyDown, className, onAnimationEnd, ...contentProps } = props;
  const { isMobile } = useDroverContext(CONTENT_NAME);
  const Component = isMobile ? Drawer.Content : PopoverContent;

  return (
    <Component
      ref={ref}
      className={cn(
        !isMobile && "min-w-60 bg-gray-1 dark:bg-black shadow-2xl p-2 border-gray-6 rounded-lg",
        className,
      )}
      align="start"
      onKeyDown={onKeyDown}
      {...contentProps}
    >
      {children}
    </Component>
  );
});

Content.displayName = CONTENT_NAME;

/* ----------------------------------------------------------------------------
 * Component Drover:Close
 * --------------------------------------------------------------------------*/

type CloseProps = PrimitiveButtonProps & {
  asChild?: boolean;
};
const CLOSE_NAME = "Close";

const Close = React.forwardRef<PrimitiveButtonElement, CloseProps>((props, ref) => {
  const { children, asChild = false, ...closeProps } = props;
  const { onOpenChange } = useDroverContext(CLOSE_NAME);
  const Comp = asChild ? Slot : "button";

  return (
    <Comp
      ref={ref}
      {...closeProps}
      onClick={(e) => {
        e.stopPropagation();
        onOpenChange(false);
      }}
    >
      {children}
    </Comp>
  );
});

Close.displayName = CLOSE_NAME;

/* ----------------------------------------------------------------------------
 * Component Drover:Nested
 * --------------------------------------------------------------------------*/

type NestedProps = PrimitiveDivProps & {
  open?: boolean;
  defaultOpen?: boolean;
  onOpenChange?: (open: boolean) => void;
};
const NESTED_NAME = "Nested";

const Nested: React.FC<NestedProps> = (props) => {
  const { children, open: openProp, onOpenChange, defaultOpen, onDrag, ...nestedProps } = props;
  const {
    isMobile,
    open: rootOpen,
    onOpenChange: onRootOpenChange,
  } = useDroverContext(NESTED_NAME);
  const NestedComponent = isMobile ? Drawer.Nested : Popover;
  const [open, setOpen] = useControllableState({
    prop: openProp,
    defaultProp: defaultOpen ?? false,
    onChange: onOpenChange,
    caller: NESTED_NAME,
  });

  return (
    <NestedComponent
      open={open}
      onOpenChange={setOpen}
      onClose={() => rootOpen && onRootOpenChange(false)}
      {...nestedProps}
    >
      {children}
    </NestedComponent>
  );
};

/* ----------------------------------------------------------------------------
 * Exports
 * --------------------------------------------------------------------------*/

export const Drover = Object.assign(
  {},
  {
    Root,
    Trigger,
    Content,
    Close,
    Nested,
  },
);
