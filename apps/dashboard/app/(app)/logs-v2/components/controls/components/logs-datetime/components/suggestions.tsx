"use client";
import { Check } from "@unkey/icons";
import { cn } from "lib/utils";
import type { PropsWithChildren } from "react";

type SuggestionOption = {
  id: number;
  value: string | number | undefined;
  display: string;
  checked: boolean;
};

export type OptionsType = SuggestionOption[];

interface SuggestionsProps extends PropsWithChildren {
  className?: string;
  options: Array<SuggestionOption>;
  onChange: (id: number) => void;
}

export const DateTimeSuggestions = ({ className, options, onChange }: SuggestionsProps) => {
  return (
    <div className={cn("flex flex-col justify-center gap-1.5", className)} role="listbox">
      {options.map(({ id, display, checked }) => (
        <div
          className="w-full px-2 rounded rounded-lg h-8 inline-flex items-center font-medium leading-4 hover:bg-gray-3"
          key={id}
        >
          <button
            type="button"
            className="w-full text-left text-sm"
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
