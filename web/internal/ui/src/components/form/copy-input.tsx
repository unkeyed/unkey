"use client";

// biome-ignore lint/style/useImportType: this package compiles JSX with the classic runtime, so React must stay a value import
import * as React from "react";
import { cn } from "../../lib/utils";
import { CopyButton } from "../buttons/copy-button";
import { InputGroup, InputGroupAddon, InputGroupInput } from "./input-group";

type CopyInputProps = Omit<React.ComponentProps<"input">, "value" | "readOnly"> & {
  value: string;
  toastMessage?: string;
};

function CopyInput({ className, value, toastMessage = value, ...props }: CopyInputProps) {
  return (
    <InputGroup data-slot="copy-input" className={cn("bg-gray-2", className)}>
      <InputGroupInput readOnly value={value} className="text-gray-11" {...props} />
      <InputGroupAddon align="inline-end">
        <CopyButton
          value={value}
          variant="ghost"
          toastMessage={toastMessage}
          className="text-gray-11 hover:text-gray-12"
        />
      </InputGroupAddon>
    </InputGroup>
  );
}

export { CopyInput, type CopyInputProps };
