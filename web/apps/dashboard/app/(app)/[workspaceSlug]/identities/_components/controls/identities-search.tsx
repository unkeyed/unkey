"use client";

import { Magnifier, XMark } from "@unkey/icons";
import { Button, InputGroup, InputGroupAddon, InputGroupInput } from "@unkey/ui";
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

  return (
    <div className="flex h-8 w-full items-center md:w-80">
      <InputGroup className="h-8">
        <InputGroupAddon className="pointer-events-none">
          <Magnifier className="text-accent-9 size-4" />
        </InputGroupAddon>
        <InputGroupInput
          aria-label="Search identities"
          type="text"
          value={search}
          maxLength={256}
          placeholder="Search identities by ID or external ID..."
          className="h-8 text-[13px] font-medium"
          onChange={(event) => setSearch(event.target.value || null)}
          onKeyDown={(event) => {
            if (event.key === "Escape") {
              setSearch(null);
            }
          }}
        />
        {search ? (
          <InputGroupAddon align="inline-end">
            <Button
              type="button"
              variant="ghost"
              size="icon"
              aria-label="Clear search"
              onClick={() => setSearch(null)}
            >
              <XMark className="size-4" />
            </Button>
          </InputGroupAddon>
        ) : null}
      </InputGroup>
    </div>
  );
};
