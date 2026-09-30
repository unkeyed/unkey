"use client";

import { IconMagnifierOutline18, IconXmarkOutline18 } from "@unkey/icons";
import { Button, InputGroup, InputGroupAddon, InputGroupInput } from "@unkey/ui";
import { useFilters } from "../../../../hooks/use-filters";

export const RootKeysSearch = () => {
  const { filters, updateFilters } = useFilters();
  const search = filters.find((filter) => filter.field === "name")?.value ?? "";

  const setSearch = (value: string) => {
    const others = filters.filter((filter) => filter.field !== "name");
    updateFilters(
      value
        ? [...others, { id: "name:contains", field: "name", operator: "contains", value }]
        : others,
    );
  };

  return (
    <div className="flex h-8 w-full items-center md:w-80">
      <InputGroup className="h-8">
        <InputGroupAddon className="pointer-events-none">
          <IconMagnifierOutline18 className="text-gray-9 size-4" />
        </InputGroupAddon>
        <InputGroupInput
          aria-label="Search root keys"
          type="text"
          value={String(search)}
          maxLength={256}
          placeholder="Search root keys by name..."
          className="h-8 text-sm font-medium"
          onChange={(event) => setSearch(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Escape") {
              setSearch("");
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
              onClick={() => setSearch("")}
            >
              <IconXmarkOutline18 className="size-4" />
            </Button>
          </InputGroupAddon>
        ) : null}
      </InputGroup>
    </div>
  );
};
