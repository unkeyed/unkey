import { cn } from "@/lib/utils";
import { Bookmark, Layers2 } from "@unkey/icons";
import { InfoTooltip, toast } from "@unkey/ui";
import { useEffect, useState } from "react";
import { useQueries } from "./queries-context";
import { QueriesItemRow } from "./queries-item-row";
import { QueriesMadeBy } from "./queries-made-by";
import { QueriesToast } from "./queries-toast";
import { getSinceTime } from "./utils";

type ListGroupProps = {
  filterList: {
    filters: Record<
      string,
      { operator: string; values: { value: string; color: string | null }[] }
    >;
    id: string;
    createdAt: number;
    bookmarked: boolean;
  };
  user?: {
    fullName: string;
    imageUrl?: string;
  };
  index: number;
  total: number;
  selectedIndex: number;
  querySelected: (index: string) => void;
  changeBookmark: (id: string) => void;
};

type ToolTipMessageType =
  | "Saved!"
  | "Save Query"
  | "Remove query from Saved"
  | "Query removed from Saved!";

const tooltipMessageOptions: { [key: string]: ToolTipMessageType } = {
  saved: "Saved!",
  save: "Save Query",
  remove: "Remove query from Saved",
  removed: "Query removed from Saved!",
};

export function ListGroup({
  filterList,
  user,
  index,
  total,
  selectedIndex,
  changeBookmark,
}: ListGroupProps) {
  const [toastMessage, setToastMessage] = useState<ToolTipMessageType>();
  const [tooltipMessage, setTooltipMessage] = useState<ToolTipMessageType>(
    filterList.bookmarked ? tooltipMessageOptions.saved : tooltipMessageOptions.save,
  );
  const { filterRowIcon, applyFilterGroup } = useQueries();

  useEffect(() => {
    if (toastMessage) {
      handleToast(toastMessage);
    }
  }, [toastMessage]);

  const handleToast = (message: ToolTipMessageType) => {
    toast.success(
      <QueriesToast message={message} undoBookmarked={() => changeBookmark(filterList.id)} />,
    );
  };

  function handleBookmarkChanged() {
    const newIsSaved = !filterList.bookmarked;
    const message = newIsSaved ? tooltipMessageOptions.saved : tooltipMessageOptions.removed;
    setToastMessage(message);
    setTooltipMessage(message);
    changeBookmark(filterList.id);
  }

  const handleMouseEnter = () => {
    setTooltipMessage(
      filterList.bookmarked ? tooltipMessageOptions.remove : tooltipMessageOptions.save,
    );
  };

  const handleSelection = () => {
    applyFilterGroup(filterList.id);
  };

  return (
    <div className="w-full">
      <div
        className={cn(
          "flex flex-row hover:bg-gray-2 cursor-pointer whitespace-nowrap rounded-[8px] pb-[9px] w-full pl-1",
          index === selectedIndex ? "bg-gray-2" : "",
        )}
      >
        <div
          className={cn("flex flex-col w-11/12 ", `tabIndex-${index}`)}
          onClick={() => handleSelection()}
          onKeyUp={(e) => e.key === "Enter"}
          tabIndex={index}
        >
          <div className=" pt-[7px] px-[8px]">
            {/* Top Row for each */}
            <div className="flex flex-row items-center justify-start h-6">
              <div className="inline-flex w-full gap-2">
                <span className="font-mono text-xs font-normal text-gray-9">from</span>
                <Layers2 className="size-3 mt-[1px]" />
                <span className="font-mono text-xs font-medium">Logs</span>
              </div>
            </div>

            {/* Filters */}
            <div className="flex flex-row w-full mt-2">
              {/* Vertical Line on Left */}
              <div className="flex flex-col ml-[9px] border-l-[1px] border-l-gray-5 w-[1px]" />
              <div className="flex flex-col gap-2 ml-0 pl-[18px] ">
                {filterList &&
                  Object.entries(filterList.filters).map(([field, filter]) => (
                    <QueriesItemRow
                      key={field}
                      list={filter.values}
                      field={field}
                      operator={filter.operator}
                      icon={filterRowIcon(field)}
                    />
                  ))}
              </div>
            </div>
            <QueriesMadeBy
              userName={user?.fullName ?? ""}
              userImageSrc={user?.imageUrl ?? ""}
              createdString={getSinceTime(filterList.createdAt)}
            />
          </div>
        </div>

        <div
          className="flex flex-col h-[24px] pr-2 mt-1.5 w-[24px]"
          onMouseEnter={handleMouseEnter}
        >
          <InfoTooltip
            variant="inverted"
            position={{ side: "top" }}
            content={tooltipMessage}
            asChild
          >
            <button
              type="button"
              className={cn(
                "flex h-7 w-6 ml-[1px]  justify-center items-center text-accent-9 rounded-md",
                filterList.bookmarked
                  ? "text-info-9 hover:bg-info-3"
                  : "hover:bg-gray-3 hover:text-accent-12",
                `tabIndex-${0}`,
              )}
              onClick={() => handleBookmarkChanged()}
              onKeyUp={(e) => e.key === "Enter"}
              aria-label={filterList.bookmarked ? "Remove from bookmarks" : "Add to bookmarks"}
            >
              <Bookmark iconSize="md-medium" filled={filterList.bookmarked} />
            </button>
          </InfoTooltip>
        </div>
      </div>
      <div
        className={cn(
          "flex flex-row bg-white dark:bg-black h-[1px] mt-[7px] mb-[8px] w-full",
          index < total - 1 && "border-b-[1px] border-b-gray-3",
        )}
      />
    </div>
  );
}
