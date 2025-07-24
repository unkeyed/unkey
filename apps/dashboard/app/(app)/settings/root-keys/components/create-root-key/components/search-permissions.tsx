"use client";
import { SearchIcon } from "@unkey/ui";
import { SearchInput } from "./search-input";

type Props = {
  isProcessing: boolean;
  search: string | undefined;
  inputRef: React.RefObject<HTMLInputElement>;
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
};
export const SearchPermissions = ({ isProcessing, search, inputRef, onChange }: Props) => {
  return (
    <div className="flex items-center gap-2 w-full flex-1 md:w-80 pl-4 py-1">
      <div className="flex-shrink-0">
        <SearchIcon isProcessing={isProcessing} />
      </div>
      <div className="flex-1">
        <SearchInput
          value={search || ""}
          placeholder="Search permissions"
          isProcessing={isProcessing}
          isLoading={false}
          loadingText="Searching..."
          clearingText="Clearing..."
          searchMode="manual"
          onChange={onChange}
          onKeyDown={() => {}}
          inputRef={inputRef}
        />
      </div>
    </div>
  );
};
