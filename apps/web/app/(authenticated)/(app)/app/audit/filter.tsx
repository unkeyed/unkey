"use client";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from "@/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Separator } from "@/components/ui/separator";
import { cn } from "@/lib/utils";
import { Check, ChevronDown } from "lucide-react";
import { parseAsArrayOf, parseAsString, useQueryState } from "nuqs";
import React from "react";
type Props = {
  options: { value: string; label: string }[];
  title: string;
  param: string;
};

export const Filter: React.FC<Props> = ({ options, title, param }) => {
  const [selected, setSelected] = useQueryState(
    param,
    parseAsArrayOf(parseAsString).withDefault([]).withOptions({
      history: "push",
      shallow: false, // otherwise server components won't notice the change
      clearOnDefault: true,
    }),
  );

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" className="flex items-center h-8 gap-2 ">
          {title}
          {selected.length > 0 && (
            <>
              <Separator orientation="vertical" className="h-4 mx-2" />
              <Badge variant="secondary" className="px-1 font-normal rounded-sm lg:hidden">
                {selected.length}
              </Badge>
              <div className="hidden space-x-1 lg:flex">
                {selected.length > 2 ? (
                  <Badge variant="secondary">{selected.length} selected</Badge>
                ) : (
                  options
                    .filter((option) => selected.includes(option.value))
                    .map((option) => (
                      <Badge variant="secondary" key={option.value}>
                        {option.label}
                      </Badge>
                    ))
                )}
              </div>
            </>
          )}
          <ChevronDown className="w-4 h-4" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[400px] p-0" align="start">
        <Command>
          <CommandInput placeholder="Events" />
          <CommandList>
            <CommandEmpty>No results found.</CommandEmpty>
            <CommandGroup>
              {options.map((option) => {
                const isSelected = selected.includes(option.value);

                return (
                  <CommandItem
                    /**
                     * We're simulating next/link behavior here and prefetching the page when they hover over an item
                     */
                    onMouseEnter={() => {
                      const copySelected = new Set(selected);
                      if (isSelected) {
                        copySelected.delete(option.value);
                      } else {
                        copySelected.add(option.value);
                      }
                      // params.prefetch(param, Array.from(copySelected).join(","));
                    }}
                    key={option.value}
                    onSelect={() => {
                      const next = isSelected
                        ? selected.filter((v) => v !== option.value)
                        : Array.from(new Set([...selected, option.value]));

                      setSelected(next);
                      // params.set(param, next);
                    }}
                  >
                    <div
                      className={cn(
                        "mr-2 flex h-4 w-4 items-center justify-center rounded-sm border border-primary",
                        isSelected
                          ? "bg-primary text-primary-foreground"
                          : "opacity-50 [&_svg]:invisible",
                      )}
                    >
                      <Check className={cn("h-4 w-4")} />
                    </div>
                    <span className="truncate text-ellipsis">{option.label}</span>
                  </CommandItem>
                );
              })}
            </CommandGroup>
            {selected.length > 0 && (
              <>
                <CommandSeparator />
                <CommandGroup>
                  <CommandItem
                    onSelect={() => {
                      setSelected([]);
                      // params.set(param, null);
                    }}
                    className="justify-center text-center"
                  >
                    Clear filters
                  </CommandItem>
                </CommandGroup>
              </>
            )}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
};
