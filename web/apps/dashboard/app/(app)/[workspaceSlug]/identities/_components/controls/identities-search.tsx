"use client";

import { Magnifier, XMark } from "@unkey/icons";
import { Button, Input } from "@unkey/ui";
import { parseAsString, useQueryState } from "nuqs";

export const IdentitiesSearch = () => {
  const [search, setSearch] = useQueryState(
    "search",
    parseAsString.withDefault("").withOptions({
      history: "replace",
      shallow: true,
      clearOnDefault: true,
    }),
  );
  const updateSearch = (value: string | null) => {
    setSearch(value).catch((error: unknown) => {
      console.error("Failed to update identity search", error);
    });
  };

  return (
    <div className="flex h-8 w-full items-center md:w-80">
      <Input
        aria-label="Search identities"
        type="text"
        value={search}
        maxLength={256}
        placeholder="Search identities by ID or external ID..."
        leftIcon={<Magnifier className="text-accent-9 size-4" />}
        rightIcon={
          search ? (
            <Button
              type="button"
              variant="ghost"
              size="icon"
              aria-label="Clear search"
              onClick={() => updateSearch(null)}
            >
              <XMark className="size-4" />
            </Button>
          ) : null
        }
        className="h-8 text-[13px] font-medium"
        onChange={(event) => updateSearch(event.target.value || null)}
        onKeyDown={(event) => {
          if (event.key === "Escape") {
            updateSearch(null);
          }
        }}
      />
    </div>
  );
};
