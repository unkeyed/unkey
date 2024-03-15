"use client";

import { CornerDownLeft, X } from "lucide-react";
import * as React from "react";

import { Badge } from "@/components/ui/badge";
import { Command } from "@/components/ui/command";
import { Command as CommandPrimitive } from "cmdk";

type Props = {
  title?: string;
  placeholder?: string;
  selected: string[];
  setSelected: (v: string[]) => void;
};

export const ArrayInput: React.FC<Props> = ({ title, placeholder, selected, setSelected }) => {
  const inputRef = React.useRef<HTMLInputElement>(null);
  const [inputValue, setInputValue] = React.useState("");

  const handleUnselect = (o: string) => {
    setSelected(selected.filter((s) => s !== o));
  };

  const handleKeyDown = React.useCallback(
    (e: React.KeyboardEvent<HTMLDivElement>) => {
      const input = inputRef.current;
      if (input) {
        if (e.key === "Delete" || e.key === "Backspace") {
          if (input.value === "") {
            setSelected(selected.slice(0, -1));
          }
        }
        if (e.key === "Enter" && input.value !== "") {
          console.log(selected, input.value);
          setSelected(Array.from(new Set([...selected, input.value])));
          setInputValue("");
        }
        // This is not a default behaviour of the <input /> field
        if (e.key === "Escape") {
          input.blur();
        }
      }
    },
    [selected],
  );

  return (
    <Command onKeyDown={handleKeyDown} className="overflow-visible bg-transparent">
      <div className="flex items-center h-8 p-1 text-sm border rounded-md group focus-within:border-primary">
        <div className="flex flex-wrap items-center w-full gap-1 px-2">
          {title ? <span className="mr-1 text-xs font-medium">{title}:</span> : null}
          {selected.map((o) => {
            return (
              <Badge key={o} variant="secondary">
                {o}
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
            placeholder={placeholder}
            className="flex-1 w-full bg-transparent outline-none placeholder:text-content-subtle"
          />
        </div>
        <button
          type="button"
          className="inline-flex items-center rounded-md border px-2 py-0.5 text-xs border-border bg-secondary text-secondary-foreground hover:bg-secondary/20 font-normal"
          onClick={() => {
            if (inputValue !== "") {
              setSelected(Array.from(new Set([...selected, inputValue])));
              setInputValue("");
            }
          }}
        >
          <CornerDownLeft className="w-3 h-4" />
        </button>
      </div>
    </Command>
  );
};
