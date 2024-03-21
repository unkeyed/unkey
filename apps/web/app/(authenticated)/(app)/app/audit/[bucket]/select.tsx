"use client";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { parseAsString, useQueryState } from "nuqs";
import React from "react";
type Props = {
  options: { value: string; label: string }[];
  title: string;
  param: string;
};

export const FilterSingle: React.FC<Props> = ({ options, title, param }) => {
  const [selected, setSelected] = useQueryState(
    param,
    parseAsString.withOptions({
      history: "push",
      shallow: false, // otherwise server components won't notice the change
      clearOnDefault: true,
    }),
  );

  return (
    <div>
      <Select onValueChange={setSelected} value={selected ?? undefined}>
        <SelectTrigger>
          <SelectValue placeholder={title} />
        </SelectTrigger>
        <SelectContent>
          {options.map((option) => (
            <SelectItem key={option.value} value={option.value}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
};
