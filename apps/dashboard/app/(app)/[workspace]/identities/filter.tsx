"use client";

import { Search } from "lucide-react";
import { parseAsString, useQueryState } from "nuqs";

export const SearchField: React.FC = () => {
  const [_, setSearch] = useQueryState(
    "search",
    parseAsString.withDefault("").withOptions({
      history: "push",
      shallow: false, // otherwise server components won't notice the change
      clearOnDefault: true,
    }),
  );
  return (
    <div className="border-border focus-within:border-primary/40 flex h-8 flex-grow items-center gap-2 rounded-md border bg-background px-3 py-2 text-sm">
      <Search className="h-4 w-4" />
      <input
        className="placeholder:text-content-subtle flex-grow bg-transparent focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50 "
        placeholder="Search.."
        onChange={(e) => {
          setSearch(e.currentTarget.value);
        }}
      />
    </div>
  );
};
