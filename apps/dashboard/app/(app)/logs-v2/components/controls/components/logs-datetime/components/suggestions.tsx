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
