"use client";

import { Radio as RadioPrimitive } from "@base-ui/react/radio";
import { RadioGroup as RadioGroupPrimitive } from "@base-ui/react/radio-group";
import * as React from "react";
import { cn } from "../lib/utils";

export function RadioGroup<Value>({ className, ...props }: RadioGroupPrimitive.Props<Value>) {
  return (
    <RadioGroupPrimitive
      className={(state) =>
        cn("grid gap-2", typeof className === "function" ? className(state) : className)
      }
      {...props}
    />
  );
}

export type RadioCardProps<Value> = Omit<RadioPrimitive.Root.Props<Value>, "children" | "title"> & {
  title: React.ReactNode;
  description?: React.ReactNode;
  icon?: React.ReactNode;
};

export function RadioCard<Value>({
  title,
  description,
  icon,
  className,
  ...props
}: RadioCardProps<Value>) {
  const id = React.useId();
  return (
    <RadioPrimitive.Root
      aria-labelledby={`${id}-title`}
      aria-describedby={description ? `${id}-description` : undefined}
      className={(state) =>
        cn(
          "group flex cursor-pointer items-start gap-3 rounded-lg border px-3 py-2.5 text-left transition-colors duration-150 ease-out hover:bg-grayA-2 focus:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-7 focus-visible:ring-offset-2 focus-visible:ring-offset-background data-checked:border-grayA-8 data-checked:bg-grayA-2 data-disabled:pointer-events-none data-disabled:opacity-50",
          typeof className === "function" ? className(state) : className,
        )
      }
      {...props}
    >
      {icon ? <span className="flex h-5 shrink-0 items-center text-gray-11">{icon}</span> : null}
      <span className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span id={`${id}-title`} className="text-sm font-medium text-gray-12">
          {title}
        </span>
        {description ? (
          <span id={`${id}-description`} className="text-xs text-gray-11">
            {description}
          </span>
        ) : null}
      </span>
      <RadioDot className="mt-0.5" />
    </RadioPrimitive.Root>
  );
}

export type RadioOptionProps<Value> = Omit<RadioPrimitive.Root.Props<Value>, "children"> & {
  label: React.ReactNode;
};

export function RadioOption<Value>({ label, className, ...props }: RadioOptionProps<Value>) {
  return (
    <RadioPrimitive.Root
      className={(state) =>
        cn(
          "group inline-flex cursor-pointer items-center gap-2 rounded-sm text-sm text-gray-12 focus:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-7 focus-visible:ring-offset-2 focus-visible:ring-offset-background data-disabled:cursor-not-allowed data-disabled:opacity-50",
          typeof className === "function" ? className(state) : className,
        )
      }
      {...props}
    >
      <RadioDot />
      {label}
    </RadioPrimitive.Root>
  );
}

function RadioDot({ className }: { className?: string }) {
  return (
    <span
      className={cn(
        "flex size-4 shrink-0 items-center justify-center rounded-full border border-input transition-colors duration-150 ease-out group-data-checked:border-gray-12",
        className,
      )}
    >
      <RadioPrimitive.Indicator className="size-2 rounded-full bg-gray-12" />
    </span>
  );
}
