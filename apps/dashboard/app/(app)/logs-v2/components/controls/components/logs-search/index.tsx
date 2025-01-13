import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { Magnifier } from "@unkey/icons";
import { SearchPopover } from "./components/search-popover";

export const LogsSearch = () => {
  return (
    <SearchPopover>
      <div className="group">
        <Button
          variant="ghost"
          className={cn(
            "group-data-[state=open]:bg-gray-4 px-2 group-data-[state=open]:border-gray-12 group-data-[state=open]:border-2"
          )}
          aria-label="Search logs"
          aria-haspopup="true"
          title="Press 'S' to toggle filters"
        >
          <Magnifier className="text-accent-9 size-4" />
          <span className="text-accent-12 font-medium text-[13px]">
            Search and filter with AI…
          </span>
        </Button>
      </div>
    </SearchPopover>
  );
};
