"use client";

// biome-ignore lint/style/useImportType: this package compiles JSX with the classic runtime, so React must stay a value import
import * as React from "react";
import { CopyButton } from "../buttons/copy-button";
import { InputGroup, InputGroupAddon, InputGroupInput } from "./input-group";

type CopyInputProps = Omit<React.ComponentProps<"input">, "value" | "readOnly"> & {
  value: string;
  toastMessage?: string;
};

function CopyInput({ className, value, toastMessage = value, ...props }: CopyInputProps) {
  return (
    <InputGroup data-slot="copy-input" className={className}>
      <InputGroupInput readOnly value={value} className="text-gray-11" {...props} />
      <InputGroupAddon align="inline-end">
        <CopyButton value={value} variant="ghost" toastMessage={toastMessage} />
      </InputGroupAddon>
    </InputGroup>
  );
}

export { CopyInput, type CopyInputProps };
