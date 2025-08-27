"use client";
import { SearchIcon } from "@unkey/ui";
import type { ChangeEvent, RefObject } from "react";
import { ROOT_KEY_MESSAGES } from "../constants";
import { SEARCH_MODES, SearchInput } from "./search-input";

type Props = {
  isProcessing: boolean;
  search: string | undefined;
  inputRef: RefObject<HTMLInputElement>;
  onChange: (e: ChangeEvent<HTMLInputElement>) => void;
};
const SearchPermissions = ({ isProcessing, search, inputRef, onChange }: Props) => {
  const isSearching = isProcessing && (search?.trim().length ?? 0) > 0;

  return (
    <div className="flex flex-row items-center gap-2 w-full md:w-[calc(100%_-_16px)] pl-4 py-1 rounded-lg">
      <div className="flex-shrink-0">
        <SearchIcon isProcessing={isProcessing} />
      </div>
      <div className="flex w-full">
        <SearchInput
          value={search ?? ""}
          placeholder={ROOT_KEY_MESSAGES.UI.SEARCH_PERMISSIONS}
          isProcessing={isProcessing}
          isLoading={isSearching}
          loadingText="Searching..."
          clearingText="Clearing..."
          searchMode={SEARCH_MODES.MANUAL}
          onChange={onChange}
          inputRef={inputRef}
        />
      </div>
    </div>
  );
};

SearchPermissions.displayName = "SearchPermissions";

export { SearchPermissions };
