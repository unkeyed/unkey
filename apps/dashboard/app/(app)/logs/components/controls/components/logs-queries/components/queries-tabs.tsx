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
    <div className="flex mt-2 h-[45px] flex-row justify-center items-center h-10 w-full border-b-[1px] border-gray-6 p-0 m-0 gap-2">
      <Button
        variant="ghost"
        className={cn(
          "h-full bg-base-12 rounded-b-none w-full ml-0 pl-[10px]",
          selectedTab === 1 ? "bg-accent-3" : "",
        )}
        aria-label="Log queries"
        aria-haspopup="true"
        title="Press 'Q' to toggle queries"
        onClick={() => handleSelection(1)}
      >
        <ClockRotateClockwise size="md-regular" className="text-gray-9 py-[1px]" />

        <div className="w-full">Recent</div>
        <div
          className={cn(
            "absolute bottom-0 w-full bg-transparent w-full h-[2px] pb-0 mb-0 ml-[2px]",
            selectedTab === 1 ? "bg-accent-12" : "",
          )}
        />
      </Button>
      <Button
        variant="ghost"
        className={cn(
          "h-full bg-base-12 rounded-b-none w-full",
          selectedTab === 2 ? "bg-accent-3" : "",
        )}
        aria-label="Log queries"
        aria-haspopup="true"
        title="Press 'Q' to toggle queries"
        onClick={() => handleSelection(2)}
      >
        <div className="text-gray-9 h-4 w-4">
          <Bookmark size="sm-regular" className="text-gray-9 py-[1.5px]" />
        </div>
        <div className="w-full">Saved</div>
        <div
          className={cn(
            "absolute bottom-0 w-full bg-transparent w-full h-[2px] pb-0 mb-0 ",
            selectedTab === 2 ? "bg-accent-12" : "",
          )}
        />
      </Button>
    </div>
  );
};
