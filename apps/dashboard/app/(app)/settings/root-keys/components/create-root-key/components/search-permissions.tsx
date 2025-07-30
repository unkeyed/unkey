"use client";
import { Button, InfoTooltip, SearchIcon } from "@unkey/ui";
import { SearchInput } from "./search-input";
import { XMark } from "@unkey/icons";

type Props = {
  isProcessing: boolean;
  search: string | undefined;
  inputRef: React.RefObject<HTMLInputElement>;
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
};
export const SearchPermissions = ({ isProcessing, search, inputRef, onChange }: Props) => {
  return (
    <div className="flex items-center gap-2 w-full md:w-[calc(100%-16px)] pl-4 py-1 rounded-lg mr-12">
      <div className="flex-shrink-0">
        <SearchIcon isProcessing={isProcessing} />
      </div>
      <div className="flex w-full">
        <SearchInput
          className="focus:ring-0 focus:outline-none focus:!bg-grayA-4 w-full"
          value={search || ""}
          placeholder="Search permissions"
          isProcessing={isProcessing}
          isLoading={false}
          loadingText="Searching..."
          clearingText="Clearing..."
          searchMode="manual"
          onChange={onChange}
          onKeyDown={() => { }}
          inputRef={inputRef}
        />
      </div>
      <div className="absolute right-2">
        <InfoTooltip
          content="Clear search"
        >
          <Button variant="ghost" size="icon" onClick={() => onChange({ target: { value: "" } } as React.ChangeEvent<HTMLInputElement>)} className="hover:bg-grayA-3 focus:ring-0 rounded-full">
            <XMark className="h-4 w-4" />
          </Button>
        </InfoTooltip>
      </div>
    </div>
  );
};
