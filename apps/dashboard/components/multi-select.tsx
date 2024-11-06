"use client";

import { X } from "lucide-react";
import * as React from "react";

import { Badge } from "@/components/ui/badge";
import { Command, CommandGroup, CommandItem } from "@/components/ui/command";
import { Command as CommandPrimitive } from "cmdk";

type Option = {
  label: string;
  value: string;
};

type Props = {
  options: Option[];
  placeholder?: string;
  selected: Option[];
  setSelected: React.Dispatch<React.SetStateAction<Option[]>>;
};

function deduplicate(options: Option[]): Option[] {
  const seen = new Set<string>();
  return options.filter((o) => {
    if (seen.has(o.value)) {
      return false;
    }
    seen.add(o.value);
    return true;
  });
}

export const MultiSelect: React.FC<Props> = ({ options, placeholder, selected, setSelected }) => {
  const inputRef = React.useRef<HTMLInputElement>(null);
  const [open, setOpen] = React.useState(false);
  const [inputValue, setInputValue] = React.useState("");

  const handleUnselect = (o: Option) => {
    setSelected((prev) => prev.filter((s) => s.value !== o.value));
  };

  const handleKeyDown = React.useCallback(
    (e: React.KeyboardEvent<HTMLDivElement>) => {
      const input = inputRef.current;
      if (input) {
        if (e.key === "Delete" || e.key === "Backspace") {
          if (input.value === "") {
            setSelected((prev) => {
              const newSelected = [...prev];
              newSelected.pop();
              return newSelected;
            });
          }
        }

        // This is not a default behaviour of the <input /> field
        if (e.key === "Escape") {
          input.blur();
        }
      }
    },
    [setSelected],
  );

  const selectables = options.filter((o) => !selected.includes(o));

  return (
    <Command onKeyDown={handleKeyDown} className="overflow-visible bg-transparent">
      <div className="flex items-center p-1 text-sm border rounded-md min-h-8 group focus-within:border-primary">
        <div className="flex flex-wrap w-full gap-1 ">
          {selected.map((o) => {
            return (
              <Badge key={o.value} variant="secondary">
                {o.label}
                <button
                  type="button"
                  className="ml-1 rounded-full outline-none"
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      handleUnselect(o);
                    }
                  }}
                  onMouseDown={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                  }}
                  onClick={() => handleUnselect(o)}
                >
                  <X className="w-3 h-3 text-content-muted hover:text-content" />
                </button>
              </Badge>
            );
          })}
          {/* Avoid having the "Search" Icon */}
          <CommandPrimitive.Input
            ref={inputRef}
            value={inputValue}
            onValueChange={setInputValue}
            onBlur={() => setOpen(false)}
            onFocus={() => setOpen(true)}
            placeholder={placeholder}
            className="flex-1 w-full bg-transparent outline-none placeholder:text-content-subtle"
          />
        </div>
      </div>
      <div className="relative mt-2">
        {open && selectables.length > 0 ? (
          <div className="absolute top-0 z-10 w-full border rounded-md shadow-md outline-none bg-background-subtle text-content animate-in">
            <CommandGroup className="h-full overflow-auto">
              {selectables
                .filter((o) => !selected.some((s) => s.value === o.value))
                .map((o) => {
                  return (
                    <CommandItem
                      key={o.value}
                      onMouseDown={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                      }}
                      onSelect={(_value) => {
                        setInputValue("");
                        setSelected((prev) => {
                          return deduplicate([...prev, o]);
                        });
                      }}
                      className={"cursor-pointer"}
                    >
                      {o.label}
                    </CommandItem>
                  );
                })}
            </CommandGroup>
          </div>
        ) : null}
      </div>
    </Command>
  );
};
