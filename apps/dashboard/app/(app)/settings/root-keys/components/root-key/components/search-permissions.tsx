"use client";
import { XMark } from "@unkey/icons";
import { Button, InfoTooltip, SearchIcon } from "@unkey/ui";
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
  const handleClear = () => {
    if (inputRef.current) {
      inputRef.current.value = "";
      // Trigger a native input event so upstream onChange sees the change
      const ev = new Event("input", { bubbles: true });
      inputRef.current.dispatchEvent(ev);
    } else {
      onChange({
        target: { value: "" },
        currentTarget: { value: "" },
        type: "input",
      } as ChangeEvent<HTMLInputElement>);
    }
  };

  return (
    <div className="flex flex-row items-center gap-2 w-full md:w-[calc(100%-16px)] pl-4 py-1 rounded-lg">
      <div className="flex-shrink-0">
        <SearchIcon isProcessing={isProcessing} />
      </div>
      <div className="flex w-full">
        <SearchInput
          value={search ?? ""}
          placeholder={ROOT_KEY_MESSAGES.UI.SEARCH_PERMISSIONS}
          isProcessing={isProcessing}
          isLoading={false}
          loadingText="Searching..."
          clearingText="Clearing..."
          searchMode={SEARCH_MODES.MANUAL}
          onChange={onChange}
          inputRef={inputRef}
        />
      </div>
      <div className="justify-end flex-shrink-0">
        <InfoTooltip content={ROOT_KEY_MESSAGES.UI.CLEAR_SEARCH}>
          <Button
            variant="ghost"
            size="icon"
            type="button"
            aria-label={ROOT_KEY_MESSAGES.UI.CLEAR_SEARCH}
            onClick={handleClear}
            className="hover:bg-grayA-3 focus:ring-0 rounded-full"
          >
            <XMark className="h-4 w-4" />
          </Button>
        </InfoTooltip>
      </div>
    </div>
  );
};

SearchPermissions.displayName = "SearchPermissions";

export { SearchPermissions };
