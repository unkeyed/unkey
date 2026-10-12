"use client";

import { IconMagnifierOutline18, IconXmarkOutline12 } from "@unkey/icons";
import { cn } from "../../lib/utils";
import { Button } from "../buttons/button";
import { InputGroup, InputGroupAddon, InputGroupInput } from "./input-group";

const DEFAULT_MAX_LENGTH = 256;

type SearchInputProps = {
  value: string;
  onValueChange: (value: string) => void;
  label: string;
  placeholder: string;
  maxLength?: number;
  className?: string;
};

export function SearchInput({
  value,
  onValueChange,
  label,
  placeholder,
  maxLength = DEFAULT_MAX_LENGTH,
  className,
}: SearchInputProps) {
  return (
    <InputGroup className={cn("h-8", className)}>
      <InputGroupAddon className="pointer-events-none">
        <IconMagnifierOutline18 className="size-4 text-gray-9" />
      </InputGroupAddon>
      <InputGroupInput
        aria-label={label}
        type="search"
        value={value}
        maxLength={maxLength}
        placeholder={placeholder}
        className="h-8 text-sm font-medium [&::-webkit-search-cancel-button]:hidden"
        onChange={(event) => onValueChange(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === "Escape") {
            onValueChange("");
          }
        }}
      />
      {value ? (
        <InputGroupAddon align="inline-end" className="pr-1.5">
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label="Clear search"
            onClick={() => onValueChange("")}
            className="size-5 rounded-md p-0 text-gray-9 hover:text-gray-12 [&_svg]:size-3"
          >
            <IconXmarkOutline12 />
          </Button>
        </InputGroupAddon>
      ) : null}
    </InputGroup>
  );
}
