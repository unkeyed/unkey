import { cn } from "@/lib/utils";
import { Bookmark, ClockRotateClockwise } from "@unkey/icons";
import { Button } from "@unkey/ui";

type QueriesTabsProps = {
  selectedTab: number;
  onChange: (index: number) => void;
};
export const QueriesTabs = ({ selectedTab, onChange }: QueriesTabsProps) => {
  const handleSelection = (index: number) => {
    onChange(index);
  };
  return (
    <div className="flex flex-row justify-center items-center h-10 w-full border-b-[1px] border-gray-6 p-0 m-0 gap-[12px]">
      <Button
        variant="ghost"
        className={cn(
          "flex h-full bg-gray-3 rounded-b-none w-full",
          selectedTab === 1 ? "bg-accent-3" : "bg-gray-1",
        )}
        aria-label="Log queries"
        aria-haspopup="true"
        title="Press 'Q' to toggle queries"
        onClick={() => handleSelection(1)}
      >
        <div className="flex flex-start text-gray-9">
          <ClockRotateClockwise className="flex flex-start size-1" />
        </div>
        <div className="w-full">Recent</div>
        <div
          className={cn(
            "absolute bottom-0 w-full bg-transparent w-full h-[2px] pb-0 mb-0",
            selectedTab === 1 ? "bg-accent-12" : "bg-gray-1",
          )}
        />
      </Button>
      <Button
        variant="ghost"
        className={cn(
          "h-full bg-gray-3 rounded-b-none w-full",
          selectedTab === 2 ? "bg-accent-3" : "bg-gray-1",
        )}
        aria-label="Log queries"
        aria-haspopup="true"
        title="Press 'Q' to toggle queries"
        onClick={() => handleSelection(2)}
      >
        <div className="flex flex-start text-gray-9">
          <Bookmark className="flex flex-start size-2" />
        </div>
        <div className="w-full">Saved</div>
        <div
          className={cn(
            "absolute bottom-0 w-full bg-transparent w-full h-[2px] pb-0 mb-0",
            selectedTab === 2 ? "bg-accent-12" : "",
          )}
        />
      </Button>
    </div>
  );
};
