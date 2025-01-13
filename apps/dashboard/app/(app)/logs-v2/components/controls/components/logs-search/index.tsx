import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { Magnifier } from "@unkey/icons";
import { SearchPopover } from "./components/search-popover";
import { useState, useRef } from "react";
import { useKeyboardShortcut } from "@/app/(app)/logs-v2/hooks/use-keyboard-shortcut";

export const LogsSearch = () => {
  const [searchText, setSearchText] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  useKeyboardShortcut("s", () => {
    inputRef.current?.focus();
    inputRef.current?.click();
  });

  return (
    <SearchPopover>
      <div className="group">
        <Button
          variant="ghost"
          className={cn(
            "group-data-[state=open]:bg-gray-4 px-2 group-data-[state=open]:ring-gray-12 group-data-[state=open]:ring-2 flex items-center",
            "focus-within:ring-2 focus-within:ring-gray-12 focus-within:bg-gray-4",
            searchText.length > 0 ? "ring-2 ring-gray-12 bg-gray-4" : ""
          )}
          aria-label="Search logs"
          aria-haspopup="true"
          title="Press 'S' to toggle filters"
        >
          <Magnifier className="text-accent-9 size-4" />
          <input
            ref={inputRef}
            type="text"
            value={searchText}
            onChange={(e) => setSearchText(e.target.value)}
            placeholder="Search and filter with AI…"
            className="text-accent-12 font-medium text-[13px] bg-transparent border-none outline-none focus:ring-0 focus:outline-none placeholder:text-accent-12 selection:bg-gray-6"
          />
        </Button>
      </div>
    </SearchPopover>
  );
};
