import type { SavedFiltersGroup } from "@/app/(app)/logs/hooks/use-bookmarked-filters";
import { BookBookmark, Bookmark, ClockRotateClockwise } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";

type EmptyQueriesProps = {
  selectedTab: number;
  list: SavedFiltersGroup[];
};
export const EmptyQueries = ({ selectedTab, list }: EmptyQueriesProps) => {
  return list.length === 0 ||
    (list.filter((filter) => filter.bookmarked).length === 0 && selectedTab === 1) ? (
    <div className="flex justify-between w-full h-full p-2">
      <Empty>
        <Empty.Icon>
          {selectedTab === 0 ? (
            <ClockRotateClockwise size="2xl-thin" className="p-0 m-0 text-accent-12" />
          ) : (
            <Bookmark size="2xl-thin" className="w-full h-full p-0 m-0 text-accent-12" />
          )}
        </Empty.Icon>
        <Empty.Title>{selectedTab === 0 ? "No recent queries" : "No saved queries"}</Empty.Title>
        <Empty.Description>
          {selectedTab === 1
            ? "Query using the filters, and they will show up here"
            : "Save your recent queries and they will remain here"}
        </Empty.Description>
        <Empty.Actions>
          <a
            href="https://www.unkey.com/docs/introduction"
            target="_blank"
            rel="noopener noreferrer"
          >
            <Button>
              <BookBookmark />
              Documentation
            </Button>
          </a>
        </Empty.Actions>
      </Empty>
    </div>
  ) : null;
};
