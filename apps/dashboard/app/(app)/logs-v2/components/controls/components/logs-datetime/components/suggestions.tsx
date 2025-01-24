"use client";
import { Check } from "@unkey/icons";
import { cn } from "lib/utils";
// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import * as React from "react";

type SuggestionOption = {
  id: number;
  value: string | number | undefined; //ms value represents by display name
  display: string;
  checked: boolean;
};
export type OptionsType = SuggestionOption[];
type SuggestionsProps = {
  className?: string;
  children?: React.ReactNode;
  options: Array<SuggestionOption>;
  onChange: (id: number) => void;
};

export const DateTimeSuggestions: React.FC<SuggestionsProps> = ({
  className,
  options,
  onChange,
}) => {
  return (
    <div className={cn("flex flex-col justify-center gap-4 mt-2 p-2 ", className)}>
      {options.map(({ id, display, checked }) => (
        <div className="w-full inline-flex items-center" key={id}>
          <button
            type="button"
            className="w-full text-left text-accent-12 text-xs"
            onClick={() => {
              onChange(id);
            }}
          >
            {display}
          </button>
          {checked ? <Check className="justify-end size-3 text-gray-12/90" /> : null}
        </div>
      ))}
    </div>
  );
};
