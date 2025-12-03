"use client";

import { LLMSearch } from "@unkey/ui";
import { parseAsString, useQueryState } from "nuqs";

export const IdentitiesSearch = () => {
  const [_search, setSearch] = useQueryState(
    "search",
    parseAsString.withDefault("").withOptions({
      history: "replace",
      shallow: true,
      clearOnDefault: true,
    }),
  );

  return (
    <LLMSearch
      exampleQueries={[
        "Find identity with ID 'user_123'",
        "Show identities with external ID containing 'test'",
        "Find identities with external ID 'john@example.com'",
        "Show identities created in the last week",
      ]}
      isLoading={false}
      searchMode="allowTypeDuringSearch"
      onSearch={(query) => {
        setSearch(query);
      }}
    />
  );
};
