"use client";
import { ArrayInput } from "@/components/array-input";
import { Badge } from "@/components/ui/badge";
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
import { Button } from "@unkey/ui";
import { Check, ChevronDown } from "lucide-react";
import { parseAsArrayOf, parseAsString, useQueryState } from "nuqs";
import type React from "react";

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

  const handleSelection = (optionValue: string, isSelected: boolean) => {
    const next = isSelected
      ? selected.filter((v) => v !== optionValue)
      : Array.from(new Set([...selected, optionValue]));
    setSelected(next);
  };

  return (
    <Popover>
      <PopoverTrigger asChild>
        <div>
          <Button className="flex items-center h-8 gap-2 bg-transparent">
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
        </div>
      </PopoverTrigger>
      <PopoverContent className="w-[400px] p-0" align="start">
        <Command>
          <CommandInput placeholder="Search" />
          <CommandList>
            <CommandEmpty>No results found.</CommandEmpty>
            <CommandGroup>
              {options.map((option) => {
                const isSelected = selected.includes(option.value);
                return (
                  <Button
                    className="w-full p-0 m-0 bg-transparent border-none shadow-none outline-none text-inherit flex-none"
                    key={option.value}
                    onClick={() => handleSelection(option.value, isSelected)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter" || e.key === " ") {
                        e.preventDefault();
                        handleSelection(option.value, isSelected);
                      }
                    }}
                  >
                    <CommandItem
                      className="w-full"
                      onSelect={() => {
                        const next = isSelected
                          ? selected.filter((v) => v !== option.value)
                          : Array.from(new Set([...selected, option.value]));
                        setSelected(next);
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
                  </Button>
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

export const CustomFilter: React.FC<{ param: string; title: string }> = ({ param, title }) => {
  const [selected, setSelected] = useQueryState(
    param,
    parseAsArrayOf(parseAsString).withDefault([]).withOptions({
      history: "push",
      shallow: false, // otherwise server components won't notice the change
      clearOnDefault: true,
    }),
  );
  return (
    <div>
      <ArrayInput title={title} selected={selected} setSelected={setSelected} />
    </div>
  );
};
