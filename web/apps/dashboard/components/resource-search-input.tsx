"use client";

import { IconMagnifierOutline18, IconXmarkOutline18 } from "@unkey/icons";
import { Button, InputGroup, InputGroupAddon, InputGroupInput } from "@unkey/ui";
import { parseAsString, useQueryState } from "nuqs";
import { useEffect, useRef, useState } from "react";

const DEBOUNCE_MS = 300;
const MAX_LENGTH = 256;

// Keeps the box responsive while the URL — and the query keyed off it — only
// moves once typing pauses. Clearing and Escape skip the wait.
function useDebouncedQueryState(key: string, delay: number) {
  const [committed, setCommitted] = useQueryState(
    key,
    parseAsString.withDefault("").withOptions({
      history: "replace",
      shallow: true,
      clearOnDefault: true,
    }),
  );
  const [text, setText] = useState(committed);
  const [seen, setSeen] = useState(committed);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(
    () => () => {
      if (timer.current !== null) {
        clearTimeout(timer.current);
      }
    },
    [],
  );

  // Adopt a value the URL gained elsewhere (back/forward, a link), but never
  // over keystrokes still waiting to be written.
  if (committed !== seen) {
    setSeen(committed);
    if (timer.current === null) {
      setText(committed);
    }
  }

  const commit = (value: string) => {
    setSeen(value);
    setCommitted(value === "" ? null : value);
  };

  const cancelPending = () => {
    if (timer.current !== null) {
      clearTimeout(timer.current);
      timer.current = null;
    }
  };

  const type = (value: string) => {
    setText(value);
    cancelPending();
    timer.current = setTimeout(() => {
      timer.current = null;
      commit(value);
    }, delay);
  };

  const clear = () => {
    setText("");
    cancelPending();
    commit("");
  };

  return { text, type, clear };
}

type ResourceSearchInputProps = {
  queryKey: string;
  label: string;
  placeholder: string;
  debounceMs?: number;
};

export function ResourceSearchInput({
  queryKey,
  label,
  placeholder,
  debounceMs = DEBOUNCE_MS,
}: ResourceSearchInputProps) {
  const { text, type, clear } = useDebouncedQueryState(queryKey, debounceMs);

  return (
    <div className="flex h-8 w-full items-center md:w-80">
      <InputGroup className="h-8">
        <InputGroupAddon>
          <IconMagnifierOutline18 className="size-4 text-gray-9" />
        </InputGroupAddon>
        <InputGroupInput
          aria-label={label}
          type="text"
          value={text}
          maxLength={MAX_LENGTH}
          placeholder={placeholder}
          className="h-8 text-sm font-medium"
          onChange={(event) => type(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Escape") {
              clear();
            }
          }}
        />
        {text ? (
          <InputGroupAddon align="inline-end">
            <Button
              type="button"
              variant="ghost"
              size="icon"
              aria-label="Clear search"
              onClick={clear}
            >
              <IconXmarkOutline18 className="size-4" />
            </Button>
          </InputGroupAddon>
        ) : null}
      </InputGroup>
    </div>
  );
}
