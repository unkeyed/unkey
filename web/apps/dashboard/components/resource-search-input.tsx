"use client";

import { Magnifier, XMark } from "@unkey/icons";
import { Button, Input } from "@unkey/ui";
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
      <Input
        aria-label={label}
        type="text"
        value={text}
        maxLength={MAX_LENGTH}
        placeholder={placeholder}
        leftIcon={<Magnifier className="text-accent-9 size-4" />}
        rightIcon={
          text ? (
            <Button
              type="button"
              variant="ghost"
              size="icon"
              aria-label="Clear search"
              onClick={clear}
            >
              <XMark className="size-4" />
            </Button>
          ) : null
        }
        className="h-8 text-[13px] font-medium"
        onChange={(event) => type(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === "Escape") {
            clear();
          }
        }}
      />
    </div>
  );
}
