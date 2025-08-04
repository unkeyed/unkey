import { Sliders } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { DisplayPopover } from "./components/display-popover";

export const LogsDisplay = () => {
  return (
    <DisplayPopover>
      <div className="group">
        <Button
          variant="ghost"
          size="md"
          className={cn("group-data-[state=open]:bg-gray-4 px-2 rounded-lg")}
          aria-label="Filter logs"
          aria-haspopup="true"
          title="Press 'F' to toggle filters"
        >
          <Sliders className="text-accent-9 size-4" />
          <span className="text-accent-12 font-medium text-[13px]">Display</span>
        </Button>
      </div>
    </DisplayPopover>
  );
};
