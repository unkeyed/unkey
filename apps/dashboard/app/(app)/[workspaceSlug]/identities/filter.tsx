"use client";

import { Magnifier } from "@unkey/icons";
import { parseAsString, useQueryState } from "nuqs";
import { useEffect, useState } from "react";

export function SearchField(): JSX.Element {
  const [search, setSearch] = useQueryState(
    "search",
    parseAsString.withDefault("").withOptions({
      history: "replace",
      shallow: true,
      clearOnDefault: true,
    })
  );

  // Local draft state for immediate UI updates
  const [draft, setDraft] = useState(search ?? "");

  // Update draft when search changes (e.g., from browser back/forward)
  useEffect(() => {
    setDraft(search ?? "");
  }, [search]);

  // Debounced search update
  useEffect(() => {
    const timeoutId = setTimeout(() => {
      if (draft !== search) {
        setSearch(draft);
      }
    }, 300); // 300ms debounce

    return () => clearTimeout(timeoutId);
  }, [draft, search, setSearch]);

  return (
    <div className="border-border focus-within:border-primary/40 flex h-8 flex-grow items-center gap-2 rounded-md border bg-background px-3 py-2 text-sm">
      <Magnifier iconsize="md-medium" aria-hidden="true" />
      <input
        type="search"
        className="placeholder:text-content-subtle flex-grow bg-transparent focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
        placeholder="Search…"
        aria-label="Search identities"
        value={draft}
        onChange={(e) => {
          setDraft(e.currentTarget.value);
        }}
      />
    </div>
  );
}
