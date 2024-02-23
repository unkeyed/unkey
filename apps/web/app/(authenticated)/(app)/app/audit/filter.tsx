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
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import React, { useCallback, useState } from "react";

type Props = {
  options: { value: string; label: string }[];
  selected: string[];
  title: string;
  param: string;
};

export const Filter: React.FC<Props> = ({ selected, options, title, param }) => {
  const [selectedValues, setSelectedValues] = useState<string[]>(selected);

  const params = useModifySearchParams();

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" className="flex items-center h-8 gap-2 ">
          {title}
          {selectedValues.length > 0 && (
            <>
              <Separator orientation="vertical" className="h-4 mx-2" />
              <Badge variant="secondary" className="px-1 font-normal rounded-sm lg:hidden">
                {selectedValues.length}
              </Badge>
              <div className="hidden space-x-1 lg:flex">
                {selectedValues.length > 2 ? (
                  <Badge variant="secondary">{selectedValues.length} selected</Badge>
                ) : (
                  options
                    .filter((option) => selectedValues.includes(option.value))
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
                const isSelected = selectedValues.includes(option.value);

                return (
                  <CommandItem
                    /**
                     * We're simulating next/link behavior here and prefetching the page when they hover over an item
                     */
                    onMouseEnter={() => {
                      const copySelected = new Set(selectedValues);
                      if (isSelected) {
                        copySelected.delete(option.value);
                      } else {
                        copySelected.add(option.value);
                      }
                      params.prefetch(param, Array.from(copySelected).join(","));
                    }}
                    key={option.value}
                    onSelect={() => {
                      const next = isSelected
                        ? selectedValues.filter((v) => v !== option.value)
                        : Array.from(new Set([...selectedValues, option.value]));

                      setSelectedValues(next);
                      params.set(param, next);
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
            {selectedValues.length > 0 && (
              <>
                <CommandSeparator />
                <CommandGroup>
                  <CommandItem
                    onSelect={() => params.set(param, null)}
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

/**
 * Utility hook to modify the search params of the current URL
 */
export function useModifySearchParams() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams()!;

  const hrefWithSearchparam = useCallback(
    (name: string, value: string | string[] | null) => {
      const params = new URLSearchParams(searchParams.toString());
      params.delete(name);

      if (value === null) {
        // do nothing, we have already deleted it above
      } else if (Array.isArray(value)) {
        value.forEach((v) => {
          params.append(name, v);
        });
      } else {
        params.set(name, value);
      }
      return `${pathname}?${params.toString()}`;
    },
    [pathname, searchParams],
  );

  return {
    prefetch: (key: string, value: string | string[] | null) => {
      const href = hrefWithSearchparam(key, value);
      router.prefetch(href);
    },
    set: (key: string, value: string | string[] | null) => {
      const href = hrefWithSearchparam(key, value);
      router.push(href, { scroll: false });
    },
  };
}
