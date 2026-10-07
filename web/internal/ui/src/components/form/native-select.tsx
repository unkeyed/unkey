import { IconChevronDownOutline18 } from "@unkey/icons";
import type * as React from "react";
import { cn } from "../../lib/utils";
import { fieldBaseClasses, fieldInvalidClasses, fieldSurfaceClasses } from "./input-group";

type NativeSelectProps = Omit<React.ComponentProps<"select">, "size"> & {
  size?: "sm" | "default";
  wrapperClassName?: string;
};

function NativeSelect({
  className,
  wrapperClassName,
  size = "default",
  ...props
}: NativeSelectProps) {
  return (
    <div
      data-slot="native-select-wrapper"
      data-size={size}
      className={cn(
        "group/native-select relative w-fit has-[select:disabled]:cursor-not-allowed has-[select:disabled]:opacity-50",
        wrapperClassName,
      )}
    >
      <select
        data-slot="native-select"
        data-size={size}
        className={cn(
          "h-9 w-full min-w-0 cursor-pointer appearance-none py-2 pr-9 pl-3 data-[size=sm]:h-8 data-[size=sm]:py-1.5 disabled:pointer-events-none",
          fieldBaseClasses,
          fieldSurfaceClasses.default,
          fieldInvalidClasses,
          className,
        )}
        {...props}
      />
      <IconChevronDownOutline18
        aria-hidden="true"
        data-slot="native-select-icon"
        className="pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2 text-gray-11"
      />
    </div>
  );
}

function NativeSelectOption({ className, ...props }: React.ComponentProps<"option">) {
  return (
    <option
      data-slot="native-select-option"
      className={cn("bg-[Canvas] text-[CanvasText]", className)}
      {...props}
    />
  );
}

export { NativeSelect, NativeSelectOption };
