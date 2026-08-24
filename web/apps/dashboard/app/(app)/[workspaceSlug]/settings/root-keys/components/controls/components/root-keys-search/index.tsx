"use client";

import { Magnifier, XMark } from "@unkey/icons";
import { Button, Input } from "@unkey/ui";
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
      <Input
        aria-label="Search root keys"
        type="text"
        value={String(search)}
        maxLength={256}
        placeholder="Search root keys by name..."
        leftIcon={<Magnifier className="text-accent-9 size-4" />}
        rightIcon={
          search ? (
            <Button
              type="button"
              variant="ghost"
              size="icon"
              aria-label="Clear search"
              onClick={() => setSearch("")}
            >
              <XMark className="size-4" />
            </Button>
          ) : null
        }
        className="h-8 text-[13px] font-medium"
        onChange={(event) => setSearch(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === "Escape") {
            setSearch("");
          }
        }}
      />
    </div>
  );
};
