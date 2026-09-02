import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { IconSlidersOutline18 } from "nucleo-ui-outline-18";
import { DisplayPopover } from "./components/display-popover";

export const LogsDisplay = () => {
  return (
    <DisplayPopover>
      <Button
        variant="ghost"
        size="md"
        className={cn("data-popup-open:bg-gray-4 px-2 rounded-lg")}
        aria-label="Filter logs"
        aria-haspopup="true"
        title="Press 'F' to toggle filters"
      >
        <IconSlidersOutline18 className="text-accent-9 size-4" />
        <span className="text-accent-12 font-normal text-[13px]">Display</span>
      </Button>
    </DisplayPopover>
  );
};
