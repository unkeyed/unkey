import { Book2, Bookmark, ClockRotateClockwise } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";

type EmptyQueriesProps = {
  selectedTab: number;
  isEmpty: boolean;
};
export const EmptyQueries = ({ selectedTab, isEmpty }: EmptyQueriesProps) => {
  return isEmpty ? (
    <div className="flex items-center justify-between w-full h-full p-2 mt-[-15px]">
      <Empty>
        <Empty.Icon>
          {selectedTab === 0 ? (
            <ClockRotateClockwise
              iconSize="2xl-thin"
              className="p-0 text-accent-12"
            />
          ) : (
            <Bookmark
              iconSize="2xl-thin"
              className="w-full h-full p-0 m-0 text-accent-12"
            />
          )}
        </Empty.Icon>
        <Empty.Title className="mt-5">
          {selectedTab === 0 ? "No recent queries" : "No saved queries"}
        </Empty.Title>
        <Empty.Description className="mt-[10px]">
          {selectedTab === 1
            ? "Query using the filters, and they will show up here"
            : "Save your recent queries and they will remain here"}
        </Empty.Description>
        <Empty.Actions className="mt-[20px]">
          <a
            href="https://www.unkey.com/docs/introduction"
            target="_blank"
            rel="noopener noreferrer"
          >
            <Button
              variant="outline"
              size="md"
              className="flex items-center justify-center px-2"
            >
              <Book2 iconSize="sm-regular" className="py-[2px]" />
              Documentation
            </Button>
          </a>
        </Empty.Actions>
      </Empty>
    </div>
  ) : null;
};
